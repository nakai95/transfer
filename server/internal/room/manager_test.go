package room

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"
)

func TestManager_Join_MaxTowClients(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(mgr Manager) []Session
		roomId  string
		wantErr error
	}{
		{
			name: "1人目は正常に参加できる",
			setup: func(mgr Manager) []Session {
				return nil
			},
			roomId:  "test-room",
			wantErr: nil,
		},
		{
			name: "2人目も正常に参加できる",
			setup: func(mgr Manager) []Session {
				s, _ := mgr.Join(context.Background(), "test-room")
				return []Session{s}
			},
			roomId:  "test-room",
			wantErr: nil,
		},
		{
			name: "3人目は定員オーバーで参加できない",
			setup: func(mgr Manager) []Session {
				s1, _ := mgr.Join(context.Background(), "test-room")
				s2, _ := mgr.Join(context.Background(), "test-room")
				return []Session{s1, s2}
			},
			roomId:  "test-room",
			wantErr: ErrorRoomFull,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mgr := NewManager()
			sessions := tt.setup(mgr)
			defer func() {
				for _, s := range sessions {
					s.Close()
				}
			}()

			s, err := mgr.Join(ctx, tt.roomId)
			if s != nil {
				defer s.Close()
			}

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("Join() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("Join() unexpected error = %v", err)
			}

			if s == nil || s.Receive() == nil {
				t.Fatalf("Join() returned nil session or failed to receive")
			}
		})
	}
}

func TestSession_Publish_SameRoom(t *testing.T) {
	ctx := context.Background()
	mgr := NewManager()
	roomId := "test-room"

	s1, err := mgr.Join(ctx, roomId)
	if err != nil {
		t.Fatalf("Join() failed to join first session: %v", err)
	}
	defer s1.Close()

	s2, err := mgr.Join(ctx, roomId)
	if err != nil {
		t.Fatalf("Join() failed to join second session: %v", err)
	}
	defer s2.Close()

	wantMsg := []byte("hello")
	if err := s1.Publish(ctx, wantMsg); err != nil {
		t.Fatalf("Publish() failed to publish message: %v", err)
	}

	select {
	case received := <-s2.Receive():
		if !bytes.Equal(received, wantMsg) {
			t.Fatalf("Publish() expected message %v, got %v", wantMsg, received)
		}
	case <-time.After(time.Millisecond * 50):
		t.Fatalf("Publish() message not received in time")
	}
}

func TestSession_Publish_IsolatedRooms(t *testing.T) {
	ctx := context.Background()
	mgr := NewManager()
	roomId1 := "room-1"
	roomId2 := "room-2"

	s1, err := mgr.Join(ctx, roomId1)
	if err != nil {
		t.Fatalf("Join() failed to join first session: %v", err)
	}
	defer s1.Close()

	s2, err := mgr.Join(ctx, roomId2)
	if err != nil {
		t.Fatalf("Join() failed to join second session: %v", err)
	}
	defer s2.Close()

	_ = s1.Publish(ctx, []byte("secret for room-1"))

	select {
	case received := <-s2.Receive():
		t.Fatalf("Publish() message leaked to another room: %v", received)
	case <-time.After(time.Millisecond * 50):
		// No message received, as expected
	}
}
