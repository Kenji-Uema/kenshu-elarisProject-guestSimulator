package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/Kenji-Uema/guestEmulator/internal/app"
	"github.com/Kenji-Uema/guestEmulator/internal/config"
	"github.com/Kenji-Uema/guestEmulator/internal/tooling/log"
)

func main() {
	slog.SetDefault(log.NewLogger())

	configs, err := config.LoadConfigs()
	if err != nil {
		slog.Error("failed to load configs", "err", err)
		os.Exit(1)
	}
	machine, err := app.NewGuestRegisterMachine(configs.GuestRegisterMachineConfig, configs.ServicesConfig)
	if err != nil {
		slog.Error("failed to create booking machine", "err", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	runner := app.NewRunner(machine, 3)
	runner.Run(ctx)
}
