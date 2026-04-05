package flows

import (
	"github.com/Kenji-Uema/guestSimulator/internal/app/steps"
	"github.com/Kenji-Uema/guestSimulator/internal/app/steps/register_guest_step"
	"github.com/Kenji-Uema/guestSimulator/internal/config"
	"github.com/Kenji-Uema/guestSimulator/internal/domain"
	"github.com/Kenji-Uema/guestSimulator/internal/infra/http"
	"github.com/Kenji-Uema/guestSimulator/internal/port"
	"github.com/go-resty/resty/v2"
)

func NewGuestRegisterFlowWithState(state *domain.State, flowConfig config.GuestRegisterFlowConfig, serviceConfig config.ServicesConfig, cache port.Cache) (*Flow, error) {
	guestClient := http.NewRestyClient(serviceConfig.GuestManagerUrl, serviceConfig.GuestManagerPort)
	return buildGuestRegisterFlow(state, guestClient, cache, flowConfig.TimeBetweenStepsInSeconds).flowRunner, nil
}

type guestRegisterFlowParts struct {
	flow       config.GuestRegisterFlow
	flowRunner *Flow
}

func buildGuestRegisterFlow(state *domain.State, guestClient *resty.Client, cache port.Cache, timeBetweenStepsInSeconds int) guestRegisterFlowParts {
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

	return guestRegisterFlowParts{
		flow: flow,
		flowRunner: &Flow{
			spanName:                  "GuestRegisterFlow",
			zeroStep:                  steps.NewInitStep(state),
			firstStep:                 flow.Start,
			stateMap:                  flow.StateMap(),
			timeBetweenStepsInSeconds: timeBetweenStepsInSeconds,
		},
	}
}
