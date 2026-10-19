package gateway

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/zHolyWhite/MoonEngineAPI/internal/cache"
	"github.com/zHolyWhite/MoonEngineAPI/internal/metrics"
	"github.com/zHolyWhite/MoonEngineAPI/internal/worker"
	"github.com/zHolyWhite/MoonEngineAPI/pkg/protocol"
)

const (
	defaultModelVersion  = "default"
	defaultPromptVersion = 1
	maxTranslateText     = 4096
	maxTranslateTargets  = 8
)

type TranslateService struct {
	cache   *cache.Cache
	flight  *cache.Flight
	trans   *worker.TranslateClient
	metrics *metrics.Metrics
}

func NewTranslateService(c *cache.Cache, f *cache.Flight, trans *worker.TranslateClient, m *metrics.Metrics) *TranslateService {
	return &TranslateService{cache: c, flight: f, trans: trans, metrics: m}
}

type translateHTTPResponse struct {
	RequestID string                   `json:"requestId"`
	Source    string                   `json:"source"`
	Results   []worker.TranslateResult `json:"results"`
}

func (s *TranslateService) Handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, protocol.CodeInvalidRequest, "método no permitido", 0)
		return
	}

	var req worker.TranslateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, protocol.CodeInvalidRequest, "JSON inválido", 0)
		return
	}

	req.Source = normalizeLang(req.Source)
	req.Targets = normalizeTargets(req.Targets)

	if strings.TrimSpace(req.Text) == "" {
		writeError(w, r, protocol.CodeInvalidRequest, "text requerido", 0)
		return
	}
	if len(req.Text) > maxTranslateText {
		writeError(w, r, protocol.CodePayloadTooLarge, "text demasiado largo", 0)
		return
	}
	if len(req.Targets) == 0 {
		writeError(w, r, protocol.CodeInvalidRequest, "targets requerido", 0)
		return
	}
	if s.trans == nil {
		writeError(w, r, protocol.CodeWorkerUnavailable, "traducción no configurada", 0)
		return
	}

	scope := scopeFrom(r.Context())
	requestID := requestIDFrom(r.Context())
	start := time.Now()
	s.metrics.TranslateRequests.Add(1)

	results := make([]worker.TranslateResult, 0, len(req.Targets))
	missing := make([]string, 0, len(req.Targets))
	found := make(map[string]worker.TranslateResult)

	for _, target := range req.Targets {
		key := cacheKey(scope, req.Source, target, req.Text, req.Kind)
		if value, ok := s.cache.Get(key); ok {
			var result worker.TranslateResult
			if err := json.Unmarshal(value, &result); err == nil {
				result.Cached = true
				found[target] = result
				s.metrics.TranslateCached.Add(1)
				continue
			}
		}
		missing = append(missing, target)
	}

	if len(missing) > 0 {
		translated, err := s.translateMissing(r, requestID, scope, req, missing)
		if err != nil {
			s.metrics.TranslateFailed.Add(1)
			writeError(w, r, protocol.CodeWorkerUnavailable, err.Error(), 0)
			return
		}
		for target, result := range translated {
			found[target] = result
		}
	}

	for _, target := range req.Targets {
		results = append(results, found[target])
	}

	s.metrics.TranslateDuration.Observe(float64(time.Since(start).Milliseconds()))

	writeJSON(w, http.StatusOK, translateHTTPResponse{
		RequestID: requestID,
		Source:    req.Source,
		Results:   results,
	})
}

func (s *TranslateService) translateMissing(r *http.Request, requestID, scope string, req worker.TranslateRequest, missing []string) (map[string]worker.TranslateResult, error) {
	key := cacheKey(scope, req.Source, strings.Join(missing, ","), req.Text, req.Kind)

	value, err, shared := s.flight.Do(key, func() ([]byte, error) {
		resp, err := s.trans.Translate(r.Context(), worker.TranslateRequest{
			RequestID: requestID,
			Source:    req.Source,
			Targets:   missing,
			Text:      req.Text,
			Kind:      req.Kind,
		})
		if err != nil {
			return nil, err
		}

		payload, err := json.Marshal(resp.Results)
		if err != nil {
			return nil, err
		}
		return payload, nil
	})

	if shared {
		s.metrics.FlightShared.Add(1)
	}

	if err != nil {
		return nil, err
	}

	var results []worker.TranslateResult
	if err := json.Unmarshal(value, &results); err != nil {
		return nil, err
	}

	byTarget := make(map[string]worker.TranslateResult, len(results))
	for _, result := range results {
		if result.Error != "" || result.Target == "" {
			continue
		}
		byTarget[result.Target] = result

		itemKey := cacheKey(scope, req.Source, result.Target, req.Text, req.Kind)
		if data, err := json.Marshal(result); err == nil {
			s.cache.Set(itemKey, data)
		}
	}

	for _, target := range missing {
		if _, ok := byTarget[target]; !ok {
			byTarget[target] = worker.TranslateResult{
				Target: target,
				Error:  "translation_failed",
			}
		}
	}
	return byTarget, nil
}

func cacheKey(scope, source, target, text, kind string) string {
	hash := sha256.Sum256([]byte(strings.Join([]string{
		scope,
		defaultModelVersion,
		"prompt:" + itoa(defaultPromptVersion),
		source,
		target,
		kind,
		text,
	}, "\x1f")))
	return hex.EncodeToString(hash[:])
}

func normalizeLang(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "auto"
	}
	return value
}

func normalizeTargets(values []string) []string {
	seen := make(map[string]bool, len(values))
	targets := make([]string, 0, len(values))
	for _, value := range values {
		value = normalizeLang(value)
		if value == "" || value == "auto" || seen[value] {
			continue
		}
		seen[value] = true
		targets = append(targets, value)
	}
	sort.Strings(targets)
	if len(targets) > maxTranslateTargets {
		targets = targets[:maxTranslateTargets]
	}
	return targets
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	digits := ""
	for n > 0 {
		digits = string(rune('0'+n%10)) + digits
		n /= 10
	}
	return digits
}
