package session

import (
	"sync"
	"time"
)

type Session struct {
	ID        string
	Scope     string
	CreatedAt time.Time

	mu       sync.RWMutex
	lastSeen time.Time
	closed   bool
}

func New(scope, id string, now time.Time) *Session {
	return &Session{
		ID:        id,
		Scope:     scope,
		CreatedAt: now,
		lastSeen:  now,
	}
}

func (s *Session) Touch(now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.closed {
		s.lastSeen = now
	}
}

func (s *Session) LastSeen() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastSeen
}

func (s *Session) Close() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return false
	}
	s.closed = true
	return true
}

func (s *Session) Closed() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.closed
}
