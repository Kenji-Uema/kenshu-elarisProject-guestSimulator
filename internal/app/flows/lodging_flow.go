package flows

import (
	"context"
	"fmt"

	"github.com/Kenji-Uema/guestSimulator/internal/app/services"
	"github.com/Kenji-Uema/guestSimulator/internal/app/steps"
	"github.com/Kenji-Uema/guestSimulator/internal/app/steps/lodging_step"
	"github.com/Kenji-Uema/guestSimulator/internal/app/steps/register_guest_step"
	"github.com/Kenji-Uema/guestSimulator/internal/config"
	"github.com/Kenji-Uema/guestSimulator/internal/domain"
	"github.com/Kenji-Uema/guestSimulator/internal/infra/http"
	"github.com/Kenji-Uema/guestSimulator/internal/infra/websocket"
	"github.com/Kenji-Uema/guestSimulator/internal/port"
)

type LodgingFlow struct {
	*Flow
	RunLodgingStep steps.Step
}

func NewLodgingFlowWithState(state *domain.State, serviceConfig config.ServicesConfig, cache port.Cache, clock port.Clock,
	notificationService services.HourNotificationService) (*LodgingFlow, error) {
	cottageClient := http.NewRestyClient(serviceConfig.CottageManagerUrl, serviceConfig.CottageManagerPort)
	guestClient := http.NewRestyClient(serviceConfig.GuestManagerUrl, serviceConfig.GuestManagerPort)

	prepareStep := lodging_step.NewPrepareLodgingStep(state, clock, cottageClient, cache)
	registerStep := register_guest_step.NewRegisterGuestStep(guestClient, cache, state)
	bookStep := lodging_step.NewBookLodgingStep(state, guestClient, cache)
	runStep := lodging_step.NewRunLodgingStep(
		state,
		fmt.Sprintf("ws://%s:%d/lodging/chat", serviceConfig.GuestManagerUrl, serviceConfig.GuestManagerPort),
		cache,
		websocket.ClientFactory{},
		notificationService,
		config.DefaultLodgingFlow(),
	)
	endStep := steps.NewEndStep(state)

	return &LodgingFlow{
		Flow: &Flow{
			spanName:  "LodgingFlow",
			zeroStep:  steps.NewNoopStep(),
			firstStep: prepareStep,
			stateMap: map[steps.Step][]domain.WeightedTuple[steps.Step]{
				prepareStep:  {{Value: registerStep, Weight: 1}},
				registerStep: {{Value: bookStep, Weight: 1}},
				bookStep:     {{Value: runStep, Weight: 1}},
				runStep:      {{Value: endStep, Weight: 1}},
			},
		},
		RunLodgingStep: runStep,
	}, nil
}

func RunLodgingStayFlow(ctx context.Context, flow *LodgingFlow) error {
	if flow == nil || flow.RunLodgingStep == nil {
		return fmt.Errorf("failed to find RunLodgingStep in lodging flow")
	}

	return runStepGraph(ctx, "LodgingStayFlow", steps.NewNoopStep(), flow.RunLodgingStep, map[steps.Step][]domain.WeightedTuple[steps.Step]{
		flow.RunLodgingStep: nil,
	})
}
