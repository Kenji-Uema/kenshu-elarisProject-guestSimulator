package flows

import (
	"context"
	"fmt"

	"github.com/Kenji-Uema/guestSimulator/internal/app/services"
	"github.com/Kenji-Uema/guestSimulator/internal/app/steps"
	"github.com/Kenji-Uema/guestSimulator/internal/app/steps/lodging_step"
	"github.com/Kenji-Uema/guestSimulator/internal/config"
	"github.com/Kenji-Uema/guestSimulator/internal/domain"
	"github.com/Kenji-Uema/guestSimulator/internal/infra/websocket"
	"github.com/Kenji-Uema/guestSimulator/internal/port"
)

type LodgingFlow struct {
	*Flow
	RunLodgingStep steps.Step
}

func NewLodgingFlowWithState(state *domain.State, serviceConfig config.ServicesConfig, cache port.Cache,
	notificationService services.HourNotificationService) (*LodgingFlow, error) {
	runStep := lodging_step.NewRunLodgingStep(
		state,
		fmt.Sprintf("ws://%s:%d/lodging/chat", serviceConfig.GuestManagerUrl, serviceConfig.GuestManagerPort),
		cache,
		websocket.ClientFactory{},
		notificationService,
		config.DefaultLodgingFlow(),
	)

	return &LodgingFlow{
		Flow: &Flow{
			spanName:  "GuestJourneyLodgingFlow",
			zeroStep:  steps.NewNoopStep(),
			firstStep: runStep,
			stateMap: map[steps.Step][]domain.WeightedTuple[steps.Step]{
				runStep: nil,
			},
		},
		RunLodgingStep: runStep,
	}, nil
}

func RunLodgingStayFlow(ctx context.Context, flow *LodgingFlow) error {
	if flow == nil || flow.RunLodgingStep == nil {
		return fmt.Errorf("failed to find RunLodgingStep in lodging flow")
	}

	return runStepGraph(ctx, "GuestJourneyLodgingStayFlow", steps.NewNoopStep(), flow.RunLodgingStep, map[steps.Step][]domain.WeightedTuple[steps.Step]{
		flow.RunLodgingStep: nil,
	})
}
