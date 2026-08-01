package session

import (
	"sync"

	"golang.org/x/time/rate"
)

type PlayerSession struct {
	PlayerID string
	SrcLang  string
	DstLang  string
	AudioCh  chan []byte
	stopCh   chan struct{}
	mu       sync.RWMutex
	limiter  *rate.Limiter
	sendFn   func(playerID, text string)
}

func (s *PlayerSession) UpdateLangs(src, dst string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.SrcLang = src
	s.DstLang = dst
}

func (s *PlayerSession) Langs() (string, string) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.SrcLang, s.DstLang
}

func (s *PlayerSession) Stop() {
	close(s.stopCh)
}
