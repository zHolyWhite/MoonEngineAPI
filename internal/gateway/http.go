package gateway

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/zHolyWhite/MoonEngineAPI/pkg/protocol"
)

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, r *http.Request, code protocol.ErrorCode, message string, retryAfterMs int) {
	if retryAfterMs > 0 {
		secs := (retryAfterMs + 999) / 1000
		w.Header().Set("Retry-After", strconv.Itoa(secs))
	}
	writeJSON(w, protocol.HTTPStatus(code), protocol.ErrorResponse{
		Error: protocol.ErrorBody{
			Code:         code,
			Message:      message,
			Retryable:    protocol.IsRetryable(code),
			RetryAfterMs: retryAfterMs,
			RequestID:    requestIDFrom(r.Context()),
		},
	})
}