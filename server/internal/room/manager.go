package room

import (
	"context"
	"errors"
	"sync"
)

var (
	ErrorRoomFull = errors.New("room is full")
)

type Session interface {
	Receive() <-chan []byte
	Publish(ctx context.Context, data []byte) error
	Close() error
}

type session struct {
	r    *room
	recv chan []byte
	once sync.Once
}

func (s *session) Receive() <-chan []byte {
	return s.recv
}

func (s *session) Publish(ctx context.Context, data []byte) error {
	s.r.mu.Lock()
	defer s.r.mu.Unlock()

	for peer := range s.r.sessions {
		if peer != s {
			select {
			case peer.recv <- data:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
	return nil
}

func (s *session) Close() error {
	s.once.Do(func() {
		close(s.recv)

		if s.r.leave(s) {
			s.r.m.removeRoomIfEmpty(s.r.id)
		}
	})
	return nil
}

type Manager interface {
	Join(ctx context.Context, roomID string) (Session, error)
}

type manager struct {
	mu    sync.RWMutex
	rooms map[string]*room
}

type room struct {
	id       string
	mu       sync.Mutex
	sessions map[*session]struct{}
	m        *manager
}

func (r *room) leave(s *session) (empty bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.sessions, s)

	return len(r.sessions) == 0
}

func NewManager() Manager {
	return &manager{
		rooms: make(map[string]*room),
	}
}

func (m *manager) Join(ctx context.Context, roomID string) (Session, error) {
	r := m.getOrCreateRoom(roomID)
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.sessions) >= 2 {
		return nil, ErrorRoomFull
	}
	s := &session{
		r:    r,
		recv: make(chan []byte, 32),
	}
	r.sessions[s] = struct{}{}

	return s, nil
}

func (m *manager) getOrCreateRoom(roomID string) *room {
	m.mu.Lock()
	defer m.mu.Unlock()

	r, exists := m.rooms[roomID]
	if !exists {
		r = &room{
			id:       roomID,
			sessions: make(map[*session]struct{}),
			m:        m,
		}
		m.rooms[roomID] = r
	}
	return r
}

func (m *manager) removeRoomIfEmpty(roomID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	r, exists := m.rooms[roomID]
	if !exists {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.sessions) == 0 {
		delete(m.rooms, roomID)
	}
}
