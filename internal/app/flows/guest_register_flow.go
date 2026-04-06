package flows

import (
	"github.com/Kenji-Uema/guestSimulator/internal/app/steps"
	"github.com/Kenji-Uema/guestSimulator/internal/app/steps/register_guest_step"
	"github.com/Kenji-Uema/guestSimulator/internal/config"
	"github.com/Kenji-Uema/guestSimulator/internal/domain"
	"github.com/Kenji-Uema/guestSimulator/internal/infra/http"
	"github.com/Kenji-Uema/guestSimulator/internal/port"
)

func NewGuestRegisterFlowWithState(state *domain.State, serviceConfig config.ServicesConfig, cache port.Cache) (*Flow, error) {
	guestClient := http.NewRestyClient(serviceConfig.GuestManagerUrl, serviceConfig.GuestManagerPort)
	registerSteps := config.GuestRegisterFlowSteps{
		End:           steps.NewEndStep(state),
		RegisterGuest: register_guest_step.NewRegisterGuestStep(guestClient, cache, state),
		RetrieveGuest: register_guest_step.NewRetrieveGuestStep(guestClient, state),
	}

	flowDef := config.DefaultGuestRegisterFlow(registerSteps)

	return &Flow{
		spanName:  "GuestJourneyGuestRegisterFlow",
		zeroStep:  steps.NewNoopStep(),
		firstStep: flowDef.Start,
		stateMap:  flowDef.StateMap(),
	}, nil
}
