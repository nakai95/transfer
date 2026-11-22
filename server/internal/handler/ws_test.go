package handler

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/coder/websocket"
	"github.com/nakai95/transfer/server/internal/room"
)

func TestWSHandler_Communication(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mgr := room.NewManager()
	h := NewWSHandler(mgr)

	s := httptest.NewServer(h)
	defer s.Close()

	wsURL := "ws" + strings.TrimPrefix(s.URL, "http") + "/ws?room=test-room"

	c1, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatalf("client 1 failed to connect to websocket: %v", err)
	}
	defer c1.Close(websocket.StatusNormalClosure, "")

	c2, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatalf("client 2 failed to connect to websocket: %v", err)
	}
	defer c2.Close(websocket.StatusNormalClosure, "")

	wantMsg := []byte("hello")
	err = c1.Write(ctx, websocket.MessageBinary, wantMsg)
	if err != nil {
		t.Fatalf("client 1 failed to write message: %v", err)
	}

	_, gotMsg, err := c2.Read(ctx)
	if err != nil {
		t.Fatalf("client 2 failed to read message: %v", err)
	}

	if !bytes.Equal(gotMsg, wantMsg) {
		t.Errorf("got message %q, want %q", gotMsg, wantMsg)
	}
}

func TestWSHandler_HTTPValidation(t *testing.T) {
	t.Run("room クエリがない場合は 400 Bad Request を返す", func(t *testing.T) {
		mgr := room.NewManager()
		h := NewWSHandler(mgr)

		req := httptest.NewRequest("GET", "/ws", nil)
		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("got status %d, want %d", rec.Code, http.StatusBadRequest)
		}
	})

	t.Run("roomが満席の場合は 409 Conflict を返す", func(t *testing.T) {
		mgr := room.NewManager()
		h := NewWSHandler(mgr)

		roomID := "full-room"
		s1, _ := mgr.Join(context.Background(), roomID)
		defer s1.Close()
		s2, _ := mgr.Join(context.Background(), roomID)
		defer s2.Close()

		req := httptest.NewRequest("GET", "/ws?room="+roomID, nil)
		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusConflict {
			t.Errorf("got status %d, want %d", rec.Code, http.StatusConflict)
		}
	})

}
