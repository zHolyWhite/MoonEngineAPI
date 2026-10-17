package session

import (
	"sync"
	"time"
)

type key struct {
	scope string
	id    string
}

type Manager struct {
	sessions sync.Map
	now      func() time.Time
}

func NewManager() *Manager {
	return &Manager{now: time.Now}
}

func (m *Manager) GetOrCreate(scope, id string) (*Session, bool) {
	k := key{scope: scope, id: id}
	if v, ok := m.sessions.Load(k); ok {
		s := v.(*Session)
		s.Touch(m.now())
		return s, false
	}
	s := New(scope, id, m.now())
	actual, loaded := m.sessions.LoadOrStore(k, s)
	if loaded {
		existing := actual.(*Session)
		existing.Touch(m.now())
		return existing, false
	}
	return s, true
}

func (m *Manager) Get(scope, id string) (*Session, bool) {
	k := key{scope: scope, id: id}
	v, ok := m.sessions.Load(k)
	if !ok {
		return nil, false
	}
	s := v.(*Session)
	s.Touch(m.now())
	return s, true
}

func (m *Manager) Close(scope, id string) bool {
	k := key{scope: scope, id: id}
	v, ok := m.sessions.Load(k)
	if !ok {
		return false
	}
	closed := v.(*Session).Close()
	if closed {
		m.sessions.Delete(k)
	}
	return closed
}

func (m *Manager) Count() int {
	count := 0
	m.sessions.Range(func(_, _ any) bool {
		count++
		return true
	})
	return count
}

func (m *Manager) CloseAll() int {
	closed := 0
	m.sessions.Range(func(k, v any) bool {
		if v.(*Session).Close() {
			closed++
		}
		m.sessions.Delete(k)
		return true
	})
	return closed
}
