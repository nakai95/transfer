package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nakai95/transfer/server/internal/handler"
	"github.com/nakai95/transfer/server/internal/room"
)

func main() {
	log.SetFlags(0)

	err := run()
	if err != nil {
		log.Fatal(err)
	}
}

func run() error {
	addr := ":8080"
	if len(os.Args) >= 2 {
		addr = os.Args[1]
	}

	l, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	log.Printf("listening on ws://%v/ws", l.Addr())

	roomMgr := room.NewManager()
	mux := http.NewServeMux()
	mux.Handle("/ws", handler.NewWSHandler(roomMgr))

	s := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}
	errc := make(chan error, 1)
	go func() {
		errc <- s.Serve(l)
	}()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)
	select {
	case err := <-errc:
		log.Printf("failed to serve: %v", err)
	case sig := <-sigs:
		log.Printf("terminating: %v", sig)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	return s.Shutdown(ctx)
}
