package gateway

import (
	"log/slog"
	"net/http"

	"github.com/zHolyWhite/MoonEngineAPI/internal/auth"
	"github.com/zHolyWhite/MoonEngineAPI/internal/config"
	"github.com/zHolyWhite/MoonEngineAPI/internal/metrics"
	"github.com/zHolyWhite/MoonEngineAPI/pkg/protocol"
)

type Server struct {
	cfg       *config.Config
	auth      *auth.Authenticator
	log       *slog.Logger
	ws        *WSHandler
	translate *TranslateService
	metrics   *metrics.Metrics
}

func New(cfg *config.Config, a *auth.Authenticator, log *slog.Logger, ws *WSHandler, translate *TranslateService, m *metrics.Metrics) *Server {
	return &Server{cfg: cfg, auth: a, log: log, ws: ws, translate: translate, metrics: m}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health/live", s.live)
	mux.Handle("GET /health/ready", s.protected(http.HandlerFunc(s.ready)))
	mux.Handle("GET /v1/capabilities", s.protected(http.HandlerFunc(s.capabilities)))
	mux.Handle("POST /v1/translate", s.protected(http.HandlerFunc(s.translate.Handle)))
	mux.Handle("POST /v1/compat/translate", s.protected(http.HandlerFunc(s.translate.HandleCompat)))
	mux.Handle("GET /v1/stream", s.protected(http.HandlerFunc(s.ws.Handle)))
	mux.Handle("GET /metrics", s.protected(s.metrics.Handler()))
	return s.withRequestID(s.withRecover(mux))
}

func (s *Server) live(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) ready(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "ready",
		"workers": map[string]bool{
			"stt":       s.cfg.STTURL != "",
			"translate": s.cfg.TranslateURL != "",
		},
	})
}

func (s *Server) capabilities(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, protocol.Capabilities{
		Contract:  protocol.ContractVersion,
		Endpoints: []string{"/v1/capabilities", "/v1/translate", "/v1/stream", "/health/live", "/health/ready", "/metrics"},
		
		Audio: protocol.AudioCapabilities{
			Formats:     []string{"pcm_s16le"},
			SampleRates: []int{48000},
			Channels:    []int{1},
		},
		Limits: protocol.Limits{
			MaxBodyBytes:  s.cfg.MaxBodyBytes,
			MaxFrameBytes: s.cfg.MaxFrameBytes,
		},
	})
}
