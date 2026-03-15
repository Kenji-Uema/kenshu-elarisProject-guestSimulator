package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/Kenji-Uema/guestSimulator/internal/app"
	"github.com/Kenji-Uema/guestSimulator/internal/config"
	"github.com/Kenji-Uema/guestSimulator/internal/infra/mq"
	"github.com/Kenji-Uema/guestSimulator/internal/tooling/log"
	"github.com/Kenji-Uema/guestSimulator/internal/tooling/telemetry"
	"github.com/Kenji-Uema/guestSimulator/internal/transport/http"
)

func main() {
	slog.SetDefault(log.NewLogger())
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	configs, err := config.LoadConfigs()
	if err != nil {
		slog.ErrorContext(ctx, "failed to load configs", "err", err)
		os.Exit(1)
	}

	shutdownTelemetry, err := telemetry.Init(ctx, configs.TelemetryConfig, configs.AppConfig)
	if err != nil {
		slog.ErrorContext(ctx, "failed to init telemetry", "err", err)
		os.Exit(1)
	}
	defer func() {
		if err := shutdownTelemetry(ctx); err != nil {
			slog.ErrorContext(ctx, "failed to shutdown telemetry", "err", err)
		}
	}()

	rabbitMqClient, err := mq.NewRabbitMqConnection(ctx, configs.RabbitMqConfig)
	if err != nil {
		slog.ErrorContext(ctx, "failed to init rabbitmq", "err", err)
		os.Exit(1)
	}
	defer func() {
		if err := rabbitMqClient.Close(); err != nil {
			slog.ErrorContext(ctx, "failed to close rabbitmq connection", "err", err)
		}
	}()

	probeServer := http.StartHTTPServer(configs.ProbeConfig, configs.ServicesConfig, rabbitMqClient)
	defer http.ShutDownHTTPServer(probeServer)

	//machine, err := app.NewGuestRegisterMachine(configs.GuestRegisterMachineConfig, configs.ServicesConfig)
	//if err != nil {
	//	slog.ErrorContext(ctx, "failed to create booking machine", "err", err)
	//	os.Exit(1)
	//}
	machine, err := app.NewBookingMachine(configs.BookingMachineConfig, configs.ServicesConfig)
	if err != nil {
		slog.ErrorContext(ctx, "failed to create booking machine", "err", err)
		os.Exit(1)
	}

	runner := app.NewRunner(machine, configs.GuestRegisterMachineConfig.ConcurrencyLevel)
	runner.Run(ctx)
}
