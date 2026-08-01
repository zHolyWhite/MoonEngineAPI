package session

import (
	"sync"
	"time"

	"golang.org/x/time/rate"

	"github.com/zHolyWhite/MoonEngineAPI/internal/config"
	"github.com/zHolyWhite/MoonEngineAPI/internal/pipeline"
)

type Manager struct {
	sessions sync.Map
	cfg      *config.Config
}

func NewManager(cfg *config.Config) *Manager {
	return &Manager{cfg: cfg}
}

func (m *Manager) GetOrCreate(playerID, srcLang, dstLang string, sendFn func(string, string)) *PlayerSession {
	new := &PlayerSession{
		PlayerID: playerID,
		SrcLang:  srcLang,
		DstLang:  dstLang,
		AudioCh:  make(chan []byte, m.cfg.AudioChanCap),
		stopCh:   make(chan struct{}),
		limiter:  rate.NewLimiter(rate.Every(m.cfg.RateEvery), m.cfg.RateBurst),
		sendFn:   sendFn,
	}
	actual, loaded := m.sessions.LoadOrStore(playerID, new)
	if !loaded {
		s := actual.(*PlayerSession)
		go s.runWorker(m.cfg, pipeline.NewClient(m.cfg.FlaskBaseURL))
	}
	return actual.(*PlayerSession)
}

func (m *Manager) Remove(playerID string) {
	if v, ok := m.sessions.LoadAndDelete(playerID); ok {
		v.(*PlayerSession).Stop()
	}
}

func (m *Manager) UpdateLangs(playerID, src, dst string) bool {
	if v, ok := m.sessions.Load(playerID); ok {
		v.(*PlayerSession).UpdateLangs(src, dst)
		return true
	}
	return false
}

 var _ = time.Second
