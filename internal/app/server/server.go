package app

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ad-targeting-engine/internal/api"
	"ad-targeting-engine/internal/config"
	"ad-targeting-engine/internal/engine"
	"ad-targeting-engine/internal/listener"
	"ad-targeting-engine/internal/storage"

	"github.com/rs/zerolog/log"
)

func Run(cfg config.Config) {
	rootCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Storage
	store, err := storage.New(rootCtx, cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("init storage")
	}
	defer store.Close()

	// Engine
	eng := engine.NewEngine()
	if err := eng.BuildSnapshot(rootCtx, store); err != nil {
		log.Fatal().Err(err).Msg("initial snapshot build")
	}

	// HTTP
	h := api.NewDeliveryHandler(eng)
	r := api.Router(h)

	srv := &http.Server{
		Addr:         cfg.Server.Addr,
		Handler:      r,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 3 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Listener (LISTEN/NOTIFY)
	go listener.ListenAndRefresh(rootCtx, store, eng, cfg.Listener.Channel, cfg.Backoff())

	// Server goroutine
	go func() {
		log.Info().Str("addr", cfg.Server.Addr).Msg("http server starting")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("server crashed")
		}
	}()

	// Wait for signal
	waitForSignal()
	log.Info().Msg("shutdown...")

	// Graceful shutdown
	shCtx, shCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shCancel()
	cancel() // stop background goroutines
	_ = srv.Shutdown(shCtx)
}

func waitForSignal() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
}