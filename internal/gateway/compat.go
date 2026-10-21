package gateway

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/zHolyWhite/MoonEngineAPI/internal/worker"
	"github.com/zHolyWhite/MoonEngineAPI/pkg/protocol"
)

type compatRequest struct {
	Text    string `json:"text"`
	Source  string `json:"source"`
	Target  string `json:"target"`
	Player  string `json:"player"`
	Context string `json:"context"`
}

type compatResponse struct {
	Translated string  `json:"translated"`
	Confidence float64 `json:"confidence"`
	Provider   string  `json:"provider"`
	Cached     bool    `json:"cached"`
}

func (s *TranslateService) HandleCompat(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 64*1024)

	var in compatRequest
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, r, protocol.CodeInvalidRequest, "JSON inválido", 0)
		return
	}

	target := strings.TrimSpace(in.Target)
	if strings.TrimSpace(in.Text) == "" || target == "" || strings.EqualFold(target, "auto") {
		writeError(w, r, protocol.CodeInvalidRequest, "text y target requeridos", 0)
		return
	}
	if len(in.Text) > maxTranslateText {
		writeError(w, r, protocol.CodePayloadTooLarge, "text demasiado largo", 0)
		return
	}
	if s.trans == nil {
		writeError(w, r, protocol.CodeWorkerUnavailable, "traducción no configurada", 0)
		return
	}

	source := normalizeLang(in.Source)
	kind := strings.TrimSpace(in.Context)
	if kind == "" {
		kind = "chat"
	}

	if strings.EqualFold(source, target) {
		writeJSON(w, http.StatusOK, compatResponse{
			Translated: in.Text,
			Confidence: 1.0,
			Provider:   "passthrough",
		})
		return
	}

	scope := scopeFrom(r.Context())
	requestID := requestIDFrom(r.Context())
	start := time.Now()
	s.metrics.TranslateRequests.Add(1)

	key := cacheKey(scope, source, target, in.Text, kind)
	if value, ok := s.cache.Get(key); ok {
		var cached worker.TranslateResult
		if err := json.Unmarshal(value, &cached); err == nil && cached.Text != "" {
			s.metrics.TranslateCached.Add(1)
			s.metrics.TranslateDuration.Observe(float64(time.Since(start).Milliseconds()))
			writeJSON(w, http.StatusOK, compatResponse{
				Translated: cached.Text,
				Confidence: 1.0,
				Provider:   "moonengine",
				Cached:     true,
			})
			return
		}
	}

	results, err := s.translateMissing(r, requestID, scope, worker.TranslateRequest{
		Source: source,
		Text:   in.Text,
		Kind:   kind,
	}, []string{target})
	if err != nil {
		s.metrics.TranslateFailed.Add(1)
		writeError(w, r, protocol.CodeWorkerUnavailable, err.Error(), 0)
		return
	}

	result := results[target]
	if result.Error != "" || result.Text == "" {
		s.metrics.TranslateFailed.Add(1)
		writeError(w, r, protocol.CodeWorkerUnavailable, "traducción fallida", 0)
		return
	}

	s.metrics.TranslateDuration.Observe(float64(time.Since(start).Milliseconds()))
	writeJSON(w, http.StatusOK, compatResponse{
		Translated: result.Text,
		Confidence: 1.0,
		Provider:   "moonengine",
		Cached:     result.Cached,
	})
}