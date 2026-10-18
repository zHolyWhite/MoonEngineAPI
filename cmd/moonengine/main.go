package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/zHolyWhite/MoonEngineAPI/internal/auth"
	"github.com/zHolyWhite/MoonEngineAPI/internal/config"
	"github.com/zHolyWhite/MoonEngineAPI/internal/gateway"
	"github.com/zHolyWhite/MoonEngineAPI/internal/scheduler"
	"github.com/zHolyWhite/MoonEngineAPI/internal/session"
	"github.com/zHolyWhite/MoonEngineAPI/internal/worker"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("configuración inválida", "error", err)
		os.Exit(1)
	}

	schedCfg, err := scheduler.LoadConfig()
	if err != nil {
		slog.Error("configuración del scheduler inválida", "error", err)
		os.Exit(1)
	}

	var level slog.Level
	_ = level.UnmarshalText([]byte(cfg.LogLevel))
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level}))

	sessions := session.NewManager()
	sched := scheduler.New(schedCfg, log)
	stt := worker.NewSTTClient(cfg.STTURL, 15*time.Second)
	trans := worker.NewTranslateClient(cfg.TranslateURL, 30*time.Second)

	srv := gateway.New(cfg, auth.New(cfg.APIKeys), log, gateway.NewWSHandler(cfg, auth.New(cfg.APIKeys), sessions, sched, stt, trans, log))
	httpServer := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       90 * time.Second,
		MaxHeaderBytes:    16 * 1024,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	sched.Start(ctx)

	errCh := make(chan error, 1)
	go func() {
		log.Info("moonengine escuchando", "addr", cfg.ListenAddr)
		errCh <- httpServer.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if !errors.Is(err, http.ErrServerClosed) {
			log.Error("fallo del servidor", "error", err)
			os.Exit(1)
		}
	case <-ctx.Done():
		log.Info("apagando")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()
		sched.Close()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			log.Error("apagado forzado", "error", err)
			os.Exit(1)
		}
	}
}