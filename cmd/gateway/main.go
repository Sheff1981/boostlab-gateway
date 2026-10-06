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

	"github.com/Sheff1981/boostlab-gateway/internal/config"
	"github.com/Sheff1981/boostlab-gateway/internal/probe"
	"github.com/Sheff1981/boostlab-gateway/internal/status"
)

func main() {
	cfg := config.Load()
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	startedAt := time.Now()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	probeServer := probe.Server{Addr: cfg.UDPAddr, Log: log}
	go func() {
		log.Info("udp probe listening", "addr", cfg.UDPAddr)
		if err := probeServer.Run(ctx); err != nil {
			log.Error("udp probe stopped", "error", err)
			stop()
		}
	}()

	httpServer := &http.Server{
		Addr: cfg.HTTPAddr,
		Handler: status.New(
			cfg.NodeID,
			cfg.Region,
			cfg.UDPAddr,
			cfg.WireGuardPublicKey,
			cfg.WireGuardPort,
			startedAt,
		).Routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Info(
			"http control listening",
			"addr", cfg.HTTPAddr,
			"node_id", cfg.NodeID,
			"region", cfg.Region,
			"wireguard_port", cfg.WireGuardPort,
			"wireguard_public_key_configured", cfg.WireGuardPublicKey != "",
		)
		err := httpServer.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("http server stopped", "error", err)
			stop()
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = httpServer.Shutdown(shutdownCtx)
	log.Info("gateway stopped")
}
