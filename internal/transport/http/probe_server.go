package http

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/Kenji-Uema/guestSimulator/internal/config"
	"github.com/Kenji-Uema/guestSimulator/internal/infra/mq"
	"github.com/Kenji-Uema/guestSimulator/internal/transport/http/probe"
)

func StartHTTPServer(probeConfig config.ProbeConfig, serviceConfig config.ServicesConfig, rabbitMqClient *mq.RabbitMqConnection) *http.Server {
	cottageClient := NewRestyClient(serviceConfig.CottageManagerUrl, serviceConfig.CottageManagerPort)
	guestClient := NewRestyClient(serviceConfig.GuestManagerUrl, serviceConfig.GuestManagerPort)

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", probe.HealthHandler())
	mux.HandleFunc("/readyz", probe.ReadinessHandler(rabbitMqClient, cottageClient, guestClient))

	server := &http.Server{
		Addr:              fmt.Sprintf("%s:%d", probeConfig.Address, probeConfig.Port),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		ctx := context.Background()
		slog.InfoContext(ctx, "http health listening", "address", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.ErrorContext(ctx, "http serve", "error", err)
		}
	}()

	return server
}

func ShutDownHTTPServer(server *http.Server) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		slog.ErrorContext(ctx, "http shutdown", "error", err)
	}
}
