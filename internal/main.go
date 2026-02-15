package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/Kenji-Uema/guestEmulator/internal/app"
	"github.com/Kenji-Uema/guestEmulator/internal/config"
	"github.com/Kenji-Uema/guestEmulator/internal/tooling/log"
	"github.com/Kenji-Uema/guestEmulator/internal/tooling/telemetry"
	"github.com/Kenji-Uema/guestEmulator/internal/transport/http"
)

func main() {
	slog.SetDefault(log.NewLogger())

	baseCtx := context.Background()
	configs, err := config.LoadConfigs()
	if err != nil {
		slog.ErrorContext(baseCtx, "failed to load configs", "err", err)
		os.Exit(1)
	}

	shutdownTelemetry, err := telemetry.Init(baseCtx, configs.TelemetryConfig, configs.AppConfig)
	if err != nil {
		slog.ErrorContext(baseCtx, "failed to init telemetry", "err", err)
		os.Exit(1)
	}
	defer func() {
		if err := shutdownTelemetry(baseCtx); err != nil {
			slog.ErrorContext(baseCtx, "failed to shutdown telemetry", "err", err)
		}
	}()

	probeServer := http.StartHTTPServer(configs.ProbeConfig, configs.ServicesConfig)
	defer http.ShutDownHTTPServer(probeServer)

	machine, err := app.NewGuestRegisterMachine(configs.GuestRegisterMachineConfig, configs.ServicesConfig)
	if err != nil {
		slog.ErrorContext(baseCtx, "failed to create booking machine", "err", err)
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(baseCtx, syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	runner := app.NewRunner(machine, configs.GuestRegisterMachineConfig.ConcurrencyLevel)
	runner.Run(ctx)
}
