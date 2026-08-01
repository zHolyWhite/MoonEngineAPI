package session

import "sync"

type SessionManager struct {
	sessions sync.Map
}

func (m *SessionManager) GetOrCreate(playerID string, srcLang, dstLang string) *PlayerSession {
	s, loaded := m.sessions.LoadOrStore(playerID, newSession(playerID, srcLang, dstLang))
	if !loaded {
		go s.(*PlayerSession).runWorker()
	}
	return s.(*PlayerSession)
}

type PlayerSession struct {
	PlayerID string
	SrcLang  string
	DstLang  string
	AudioCh  chan[]byte
	stopCh 	 chan struct[]
	mu       sync.RMutex
	limiter *rate.Limiter
}

//	func (s *PlayerSession) processPipeline(audio []byte) {
//		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
//		defer cancel()

//	if !s.limiter.Allow() {
//		log.Println("[%s] rate limit, skipping utterance", s.PlayerID)
//	return
//	}
//}
