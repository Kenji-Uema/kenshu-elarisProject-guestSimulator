package app

import (
	"github.com/Kenji-Uema/guestEmulator/internal/app/state"
	"github.com/Kenji-Uema/guestEmulator/internal/app/state/register_guest_state"
	"github.com/Kenji-Uema/guestEmulator/internal/config"
	"github.com/Kenji-Uema/guestEmulator/internal/domain"
	"github.com/Kenji-Uema/guestEmulator/internal/transport/http"
)

func NewGuestRegisterMachine(machineConfig config.GuestRegisterMachineConfig, serviceConfig config.ServicesConfig) (*Machine, error) {
	guestClient := http.NewRestyClient(serviceConfig.GuestManagerUrl)

	zeroState := state.NewInitState()
	registerGuestStates := map[string]state.State{
		"End":           state.Adapter[domain.IgnoredField, domain.IgnoredField]{State: state.NewEndState()},
		"RegisterGuest": state.Adapter[domain.IgnoredField, string]{State: register_guest_state.NewRegisterGuestState(guestClient)},
		"RetrieveGuest": state.Adapter[string, domain.IgnoredField]{State: register_guest_state.NewRetrieveGuestState(guestClient)},
	}

	stateMap, err := readGraph(machineConfig.GraphFile, registerGuestStates)
	if err != nil {
		return nil, err
	}

	return &Machine{
		zeroState:                 zeroState,
		initState:                 registerGuestStates["RegisterGuest"],
		stateMap:                  stateMap,
		timeBetweenStepsInSeconds: machineConfig.TimeBetweenStepsInSeconds,
	}, nil
}
