package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/Kenji-Uema/guestEmulator/internal/app"
	"github.com/Kenji-Uema/guestEmulator/internal/config"
)

func main() {
	cfg := config.BookingMachineConfig{
		ClockEmuUrl:       "localhost:50052",
		CottageManagerUrl: "http://localhost:8080/cottages",
		GuestManagerUrl:   "http://localhost:8080/guests",
		GraphFile:         "docs/booking_mdp.dot",
	}

	machine, err := app.NewBookingMachine(cfg)
	if err != nil {
		slog.Error("failed to create booking machine", "err", err)
		os.Exit(1)
	}

	if err := machine.Start(context.Background()); err != nil {
		slog.Error("booking machine stopped with error", "err", err)
		os.Exit(1)
	}
}
