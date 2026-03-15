package app

import (
	"github.com/Kenji-Uema/guestSimulator/internal/app/steps"
	"github.com/Kenji-Uema/guestSimulator/internal/app/steps/booking_step"
	"github.com/Kenji-Uema/guestSimulator/internal/app/steps/register_guest_step"
	"github.com/Kenji-Uema/guestSimulator/internal/config"
	"github.com/Kenji-Uema/guestSimulator/internal/domain"
	"github.com/Kenji-Uema/guestSimulator/internal/transport/grpc"
	"github.com/Kenji-Uema/guestSimulator/internal/transport/http"
)

func NewBookingMachine(machineConfig config.BookingMachineConfig, serviceConfig config.ServicesConfig) (*Machine, error) {
	cottageClient := http.NewRestyClient(serviceConfig.CottageManagerUrl, serviceConfig.CottageManagerPort)
	guestClient := http.NewRestyClient(serviceConfig.GuestManagerUrl, serviceConfig.GuestManagerPort)
	clockEmu, err := grpc.NewClockEmu(serviceConfig)
	if err != nil {
		return nil, err
	}

	state := &domain.State{}

	bookingMachineStates := map[string]steps.Step{
		"End":           steps.NewEndStep(state),
		"SelectCottage": booking_step.NewSelectCottageStep(state),
		"ListCottages":  booking_step.NewListCottagesStep(state, cottageClient),
		"SelectPeriod":  booking_step.NewSelectPeriodStep(state, clockEmu, cottageClient),
		"RegisterGuest": register_guest_step.NewRegisterGuestStep(guestClient, state),
		"BookCottage":   booking_step.NewBookCottageStep(state, cottageClient),
	}

	stateMap, err := readGraph(machineConfig.GraphFile, bookingMachineStates)
	if err != nil {
		return nil, err
	}

	return &Machine{
		zeroStep:                  steps.NewInitStep(state),
		firstStep:                 bookingMachineStates["ListCottages"],
		stateMap:                  stateMap,
		timeBetweenStepsInSeconds: machineConfig.TimeBetweenStepsInSeconds,
	}, nil
}
