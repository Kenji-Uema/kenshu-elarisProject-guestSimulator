package machines

import (
	"github.com/Kenji-Uema/guestSimulator/internal/app/steps"
	"github.com/Kenji-Uema/guestSimulator/internal/app/steps/register_guest_step"
	"github.com/Kenji-Uema/guestSimulator/internal/config"
	"github.com/Kenji-Uema/guestSimulator/internal/domain"
	"github.com/Kenji-Uema/guestSimulator/internal/infra/http"
	"github.com/Kenji-Uema/guestSimulator/internal/port"
	"github.com/go-resty/resty/v2"
)

func NewGuestRegisterMachineWithState(state *domain.State, machineConfig config.GuestRegisterMachineConfig, serviceConfig config.ServicesConfig, cache port.Cache) (*Machine, error) {
	guestClient := http.NewRestyClient(serviceConfig.GuestManagerUrl, serviceConfig.GuestManagerPort)
	return buildGuestRegisterMachine(state, guestClient, cache, machineConfig.TimeBetweenStepsInSeconds).machine, nil
}

type guestRegisterMachineParts struct {
	flow    config.GuestRegisterFlow
	machine *Machine
}

func buildGuestRegisterMachine(state *domain.State, guestClient *resty.Client, cache port.Cache, timeBetweenStepsInSeconds int) guestRegisterMachineParts {
	registerGuestStates := map[string]steps.Step{
		"End":           steps.NewEndStep(state),
		"RegisterGuest": register_guest_step.NewRegisterGuestStep(guestClient, cache, state),
		"RetrieveGuest": register_guest_step.NewRetrieveGuestStep(guestClient, state),
	}

	flow := config.DefaultGuestRegisterFlow(config.GuestRegisterFlowSteps{
		End:           registerGuestStates["End"],
		RegisterGuest: registerGuestStates["RegisterGuest"],
		RetrieveGuest: registerGuestStates["RetrieveGuest"],
	})

	return guestRegisterMachineParts{
		flow: flow,
		machine: &Machine{
			spanName:                  "GuestRegisterMachine",
			zeroStep:                  steps.NewInitStep(state),
			firstStep:                 flow.Start,
			stateMap:                  flow.StateMap(),
			timeBetweenStepsInSeconds: timeBetweenStepsInSeconds,
		},
	}
}
