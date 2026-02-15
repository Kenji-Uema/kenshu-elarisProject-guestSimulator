package http

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/Kenji-Uema/guestEmulator/internal/config"
	"github.com/Kenji-Uema/guestEmulator/internal/transport/http/probe"
)

func StartHTTPServer(probeConfig config.ProbeConfig, serviceConfig config.ServicesConfig) *http.Server {
	cottageClient := NewRestyClient(serviceConfig.CottageManagerUrl, serviceConfig.CottageManagerPort)
	guestClient := NewRestyClient(serviceConfig.GuestManagerUrl, serviceConfig.GuestManagerPort)

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", probe.HealthHandler())
	mux.HandleFunc("/readyz", probe.ReadinessHandler(cottageClient, guestClient))

	server := &http.Server{
		Addr:              fmt.Sprintf("%s:%d", probeConfig.Address, probeConfig.Port),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		slog.Info("http health listening", "address", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("http serve", err)
		}
	}()

	return server
}

func ShutDownHTTPServer(server *http.Server) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		slog.Error("http shutdown", err)
	}
}
