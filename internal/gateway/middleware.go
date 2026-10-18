package gateway

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"regexp"

	"github.com/zHolyWhite/MoonEngineAPI/pkg/protocol"
)

var requestIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{8,64}$`)

func newRequestID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "00000000000000000000000000000000"
	}
	return hex.EncodeToString(b)
}

func (s *Server) withRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-Id")
		if !requestIDPattern.MatchString(id) {
			id = newRequestID()
		}
		w.Header().Set("X-Request-Id", id)
		next.ServeHTTP(w, r.WithContext(withValue(r.Context(), requestIDKey, id)))
	})
}

func (s *Server) withRecover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				s.log.Error("panic en handler", "path", r.URL.Path, "requestId", requestIDFrom(r.Context()), "panic", rec)
				writeError(w, r, protocol.CodeInternal, "error interno", 0)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func (s *Server) protected(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		scope, ok := s.auth.Authenticate(r.Header.Get("Authorization"))
		if !ok {
			w.Header().Set("WWW-Authenticate", "Bearer")
			writeError(w, r, protocol.CodeUnauthorized, "credenciales inválidas", 0)
			return
		}
		next.ServeHTTP(w, r.WithContext(withValue(r.Context(), scopeKey, scope)))
	})
}
