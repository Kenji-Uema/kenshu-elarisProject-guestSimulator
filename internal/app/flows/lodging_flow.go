package flows

import (
	"context"
	"fmt"
	"time"

	"github.com/Kenji-Uema/guestSimulator/internal/app/steps"
	"github.com/Kenji-Uema/guestSimulator/internal/app/steps/lodging_step"
	"github.com/Kenji-Uema/guestSimulator/internal/app/steps/register_guest_step"
	"github.com/Kenji-Uema/guestSimulator/internal/config"
	"github.com/Kenji-Uema/guestSimulator/internal/domain"
	"github.com/Kenji-Uema/guestSimulator/internal/infra/clock"
	"github.com/Kenji-Uema/guestSimulator/internal/infra/http"
	"github.com/Kenji-Uema/guestSimulator/internal/infra/websocket"
	"github.com/Kenji-Uema/guestSimulator/internal/port"
)

type HourNotificationService interface {
	HourNotification(ctx context.Context, timerCh chan interface{}, hour int)
	CurrentTime() (time.Time, bool)
}

func NewLodgingFlowWithState(state *domain.State, flowConfig config.LodgingFlowConfig, serviceConfig config.ServicesConfig, cache port.Cache,
	notificationService HourNotificationService) (*Flow, error) {
	cottageClient := http.NewRestyClient(serviceConfig.CottageManagerUrl, serviceConfig.CottageManagerPort)
	guestClient := http.NewRestyClient(serviceConfig.GuestManagerUrl, serviceConfig.GuestManagerPort)
	clockEmu, err := clock.NewClockEmu(serviceConfig)
	if err != nil {
		return nil, err
	}

	initStep := steps.NewInitStep(state)
	prepareStep := lodging_step.NewPrepareLodgingStep(state, clockEmu, cottageClient, cache)
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

	return &Flow{
		spanName:  "LodgingFlow",
		zeroStep:  initStep,
		firstStep: prepareStep,
		stateMap: map[steps.Step][]domain.WeightedTuple[steps.Step]{
			prepareStep:  {{Value: registerStep, Weight: 1}},
			registerStep: {{Value: bookStep, Weight: 1}},
			bookStep:     {{Value: runStep, Weight: 1}},
			runStep:      {{Value: endStep, Weight: 1}},
		},
		timeBetweenStepsInSeconds: flowConfig.TimeBetweenStepsInSeconds,
	}, nil
}

func RunLodgingStayFlow(ctx context.Context, flow *Flow, timeBetweenStepsInSeconds int) error {
	if flow == nil {
		return fmt.Errorf("failed to find RunLodgingStep in lodging flow")
	}

	var runLodgingStep steps.Step
	for step := range flow.stateMap {
		if step != nil && step.Name() == "RunLodgingStep" {
			runLodgingStep = step
			break
		}
	}
	if runLodgingStep == nil {
		return fmt.Errorf("failed to find RunLodgingStep in lodging flow")
	}

	return runStepGraph(ctx, "LodgingStayFlow", steps.NewNoopStep(), runLodgingStep, map[steps.Step][]domain.WeightedTuple[steps.Step]{
		runLodgingStep: nil,
	}, timeBetweenStepsInSeconds)
}
