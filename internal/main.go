package main

import (
	"context"
	"guestEmulator/internal/app"
	"guestEmulator/internal/config"
	"log"
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
