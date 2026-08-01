package main

import (
	"log"
	"net/http"

	"github.com/zHolyWhite/MoonEngineAPI/internal/config"
	"github.com/zHolyWhite/MoonEngineAPI/internal/gateway"
	"github.com/zHolyWhite/MoonEngineAPI/internal/session"
)

func main() {
	cfg := config.Load()

	sm := session.NewManager(cfg)
	ws := gateway.NewWSHandler(sm, cfg)
	http_h := gateway.NewHTTPHandler(sm)

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", ws.Handle)
	mux.HandleFunc("/health", http_h.Health)
	mux.HandleFunc("/session/lang", http_h.UpdateLang)

	log.Printf("[MoonEngine] listening on %s", cfg.ListenAddr)
	if err := http.ListenAndServe(cfg.ListenAddr, mux); err != nil {
		log.Fatalf("[MoonEngine] server error: %v", err)
	}
}
