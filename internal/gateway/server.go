package gateway

import (
	"log/slog"
	"net/http"

	"github.com/zHolyWhite/MoonEngineAPI/internal/auth"
	"github.com/zHolyWhite/MoonEngineAPI/internal/config"
	"github.com/zHolyWhite/MoonEngineAPI/pkg/protocol"
)

type Server struct {
	cfg  *config.Config
	auth *auth.Authenticator
	log  *slog.Logger
}

func New(cfg *config.Config, a *auth.Authenticator, log *slog.Logger) *Server {
	return &Server{cfg: cfg, auth: a, log: log}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health/live", s.live)
	mux.Handle("GET /health/ready", s.protected(http.HandlerFunc(s.ready)))
	mux.Handle("GET /v1/capabilities", s.protected(http.HandlerFunc(s.capabilities)))
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
		Endpoints: []string{"/v1/capabilities", "/health/live", "/health/ready"},
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