package handler

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/coder/websocket"
	"github.com/nakai95/transfer/server/internal/room"
)

type WSHandler struct {
	mgr room.Manager
}

func NewWSHandler(mgr room.Manager) *WSHandler {
	return &WSHandler{
		mgr: mgr,
	}
}

func (h *WSHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	roomID := r.URL.Query().Get("room")
	if roomID == "" {
		http.Error(w, "room query parameter is required", http.StatusBadRequest)
		return
	}

	session, err := h.mgr.Join(r.Context(), roomID)
	if err != nil {
		if errors.Is(err, room.ErrorRoomFull) {
			http.Error(w, "room is full", http.StatusConflict)
			return
		}
		http.Error(w, "failed to join room", http.StatusInternalServerError)
		return
	}
	defer session.Close()

	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	})
	if err != nil {
		log.Printf("failed to accept websocket: %v", err)
		return
	}
	defer c.CloseNow()

	h.handleConnection(r.Context(), c, session, r)
}

func (h *WSHandler) handleConnection(ctx context.Context, c *websocket.Conn, session room.Session, r *http.Request) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	done := make(chan struct{})
	go func() {
		defer close(done)
		defer cancel()

		h.writeLoop(ctx, c, session)
	}()

	h.readLoop(ctx, c, session, r)

	cancel()
	<-done
}

func (h *WSHandler) writeLoop(ctx context.Context, c *websocket.Conn, session room.Session) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg, open := <-session.Receive():
			if !open {
				return
			}
			if err := c.Write(ctx, websocket.MessageBinary, msg); err != nil {
				log.Printf("write error: %v", err)
				return
			}
		}
	}
}

func (h *WSHandler) readLoop(ctx context.Context, c *websocket.Conn, session room.Session, r *http.Request) {
	for {
		_, msg, err := c.Read(ctx)
		if err != nil {
			if websocket.CloseStatus(err) == websocket.StatusNormalClosure ||
				websocket.CloseStatus(err) == websocket.StatusGoingAway {
				return
			}
			log.Printf("read error from %v: %v", r.RemoteAddr, err)
			return
		}

		if err := session.Publish(ctx, msg); err != nil {
			log.Printf("publish error from %v: %v", r.RemoteAddr, err)
			return
		}
	}
}
