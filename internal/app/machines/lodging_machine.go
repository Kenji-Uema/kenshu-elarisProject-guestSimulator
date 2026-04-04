package machines

import (
	"context"
	"fmt"
	"time"

	"github.com/Kenji-Uema/guestSimulator/internal/app/steps"
	"github.com/Kenji-Uema/guestSimulator/internal/app/steps/lodging_step"
	"github.com/Kenji-Uema/guestSimulator/internal/app/steps/register_guest_step"
	"github.com/Kenji-Uema/guestSimulator/internal/config"
	"github.com/Kenji-Uema/guestSimulator/internal/domain"
	redisc "github.com/Kenji-Uema/guestSimulator/internal/infra/redis"
	"github.com/Kenji-Uema/guestSimulator/internal/transport/grpc"
	transporthttp "github.com/Kenji-Uema/guestSimulator/internal/transport/http"
)

type HourNotificationService interface {
	HourNotification(ctx context.Context, timerCh chan interface{}, hour int)
	CurrentTime() (time.Time, bool)
}

func NewLodgingMachine(machineConfig config.LodgingMachineConfig, serviceConfig config.ServicesConfig, redis *redisc.Redis,
	notificationService HourNotificationService) (*Machine, error) {
	return NewLodgingMachineWithState(&domain.State{}, machineConfig, serviceConfig, redis, notificationService)
}

func NewLodgingMachineWithState(state *domain.State, machineConfig config.LodgingMachineConfig, serviceConfig config.ServicesConfig, redis *redisc.Redis,
	notificationService HourNotificationService) (*Machine, error) {
	cottageClient := transporthttp.NewRestyClient(serviceConfig.CottageManagerUrl, serviceConfig.CottageManagerPort)
	guestClient := transporthttp.NewRestyClient(serviceConfig.GuestManagerUrl, serviceConfig.GuestManagerPort)
	clockEmu, err := grpc.NewClockEmu(serviceConfig)
	if err != nil {
		return nil, err
	}

	initStep := steps.NewInitStep(state)
	prepareStep := lodging_step.NewPrepareLodgingStep(state, clockEmu, cottageClient, redis)
	registerStep := register_guest_step.NewRegisterGuestStep(guestClient, redis, state)
	bookStep := lodging_step.NewBookLodgingStep(state, guestClient, redis)
	runStep := lodging_step.NewRunLodgingStep(
		state,
		fmt.Sprintf("ws://%s:%d/lodging/chat", serviceConfig.GuestManagerUrl, serviceConfig.GuestManagerPort),
		redis,
		notificationService,
		config.DefaultLodgingFlow(),
	)
	endStep := steps.NewEndStep(state)

	return &Machine{
		spanName:  "LodgingMachine",
		zeroStep:  initStep,
		firstStep: prepareStep,
		stateMap: map[steps.Step][]domain.WeightedTuple[steps.Step]{
			prepareStep:  {{Value: registerStep, Weight: 1}},
			registerStep: {{Value: bookStep, Weight: 1}},
			bookStep:     {{Value: runStep, Weight: 1}},
			runStep:      {{Value: endStep, Weight: 1}},
		},
		timeBetweenStepsInSeconds: machineConfig.TimeBetweenStepsInSeconds,
	}, nil
}

func RunLodgingStayJourney(ctx context.Context, machine *Machine, timeBetweenStepsInSeconds int) error {
	if machine == nil {
		return fmt.Errorf("failed to find RunLodgingStep in lodging machine")
	}

	var runLodgingStep steps.Step
	for step := range machine.stateMap {
		if step != nil && step.Name() == "RunLodgingStep" {
			runLodgingStep = step
			break
		}
	}
	if runLodgingStep == nil {
		return fmt.Errorf("failed to find RunLodgingStep in lodging machine")
	}

	return runStepGraph(ctx, "LodgingStayFlow", noopStep{}, runLodgingStep, map[steps.Step][]domain.WeightedTuple[steps.Step]{
		runLodgingStep: nil,
	}, timeBetweenStepsInSeconds)
}
