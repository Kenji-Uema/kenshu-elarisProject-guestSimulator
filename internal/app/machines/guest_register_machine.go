package machines

import (
	"github.com/Kenji-Uema/guestSimulator/internal/app/steps"
	"github.com/Kenji-Uema/guestSimulator/internal/app/steps/register_guest_step"
	"github.com/Kenji-Uema/guestSimulator/internal/config"
	"github.com/Kenji-Uema/guestSimulator/internal/domain"
	redisc "github.com/Kenji-Uema/guestSimulator/internal/infra/redis"
	"github.com/Kenji-Uema/guestSimulator/internal/transport/http"
	"github.com/go-resty/resty/v2"
)

func NewGuestRegisterMachine(machineConfig config.GuestRegisterMachineConfig, serviceConfig config.ServicesConfig) (*Machine, error) {
	return NewGuestRegisterMachineWithState(&domain.State{}, machineConfig, serviceConfig, nil)
}

func NewGuestRegisterMachineWithState(state *domain.State, machineConfig config.GuestRegisterMachineConfig, serviceConfig config.ServicesConfig, redis *redisc.Redis) (*Machine, error) {
	guestClient := http.NewRestyClient(serviceConfig.GuestManagerUrl, serviceConfig.GuestManagerPort)
	return buildGuestRegisterMachine(state, guestClient, redis, machineConfig.TimeBetweenStepsInSeconds).machine, nil
}

type guestRegisterMachineParts struct {
	flow    config.GuestRegisterFlow
	machine *Machine
}

func buildGuestRegisterMachine(state *domain.State, guestClient *resty.Client, redis *redisc.Redis, timeBetweenStepsInSeconds int) guestRegisterMachineParts {
	registerGuestStates := map[string]steps.Step{
		"End":           steps.NewEndStep(state),
		"RegisterGuest": register_guest_step.NewRegisterGuestStep(guestClient, redis, state),
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
