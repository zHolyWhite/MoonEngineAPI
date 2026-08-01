package gateway

import (
	"encoding/json"
	"net/http"
	"runtime"

	"github.com/zHolyWhite/MoonEngineAPI/internal/session"
)

type HTTPHandler struct {
	sm *session.Manager
}

func NewHTTPHandler(sm *session.Manager) *HTTPHandler {
	return &HTTPHandler{sm: sm}
}

func (h *HTTPHandler) Health(w http.ResponseWriter, r *http.Request) {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"status":     "healthy",
		"goroutines": runtime.NumGoroutine(),
		"alloc_mb":   ms.Alloc / 1024 / 1024,
	})
}

type updateLangRequest struct {
	PlayerID string `json:"player_id"`
	SrcLang  string `json:"src_lang"`
	DstLang  string `json:"dst_lang"`
}

func (h *HTTPHandler) UpdateLang(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req updateLangRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if req.PlayerID == "" || req.SrcLang == "" || req.DstLang == "" {
		http.Error(w, "player_id, src_lang, dst_lang required", http.StatusBadRequest)
		return
	}
	if !h.sm.UpdateLangs(req.PlayerID, req.SrcLang, req.DstLang) {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
