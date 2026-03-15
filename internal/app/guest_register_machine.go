package app

import (
	"github.com/Kenji-Uema/guestSimulator/internal/app/steps"
	"github.com/Kenji-Uema/guestSimulator/internal/app/steps/register_guest_step"
	"github.com/Kenji-Uema/guestSimulator/internal/config"
	"github.com/Kenji-Uema/guestSimulator/internal/domain"
	"github.com/Kenji-Uema/guestSimulator/internal/transport/http"
)

func NewGuestRegisterMachine(machineConfig config.GuestRegisterMachineConfig, serviceConfig config.ServicesConfig) (*Machine, error) {
	guestClient := http.NewRestyClient(serviceConfig.GuestManagerUrl, serviceConfig.GuestManagerPort)

	state := &domain.State{}

	registerGuestStates := map[string]steps.Step{
		"End":           steps.NewEndStep(state),
		"RegisterGuest": register_guest_step.NewRegisterGuestStep(guestClient, state),
		"RetrieveGuest": register_guest_step.NewRetrieveGuestStep(guestClient, state),
	}

	stateMap, err := readGraph(machineConfig.GraphFile, registerGuestStates)
	if err != nil {
		return nil, err
	}

	return &Machine{
		zeroStep:                  steps.NewInitStep(state),
		firstStep:                 registerGuestStates["RegisterGuest"],
		stateMap:                  stateMap,
		timeBetweenStepsInSeconds: machineConfig.TimeBetweenStepsInSeconds,
	}, nil
}
