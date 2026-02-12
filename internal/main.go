package main

import (
	"context"
	"log"

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
		log.Fatal(err)
	}

	if err := machine.Start(context.Background()); err != nil {
		log.Fatal(err)
	}
}
