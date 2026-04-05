package machines

import (
	"context"
	"log/slog"
	"time"

	"github.com/Kenji-Uema/guestSimulator/internal/app/services"
	"github.com/Kenji-Uema/guestSimulator/internal/app/steps"
	"github.com/Kenji-Uema/guestSimulator/internal/domain"
	"github.com/Kenji-Uema/guestSimulator/internal/infra/telemetry"
)

type Machine struct {
	spanName                  string
	zeroStep                  steps.Step
	firstStep                 steps.Step
	stateMap                  map[steps.Step][]domain.WeightedTuple[steps.Step]
	timeBetweenStepsInSeconds int
}

type Startable interface {
	Start(ctx context.Context) error
}

func (m *Machine) Start(ctx context.Context) error {
	return runStepGraph(ctx, m.spanName, m.zeroStep, m.firstStep, m.stateMap, m.timeBetweenStepsInSeconds)
}

func runStepGraph(ctx context.Context, spanName string, zeroStep steps.Step, firstStep steps.Step,
	stateMap map[steps.Step][]domain.WeightedTuple[steps.Step], timeBetweenStepsInSeconds int) error {
	machineCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	machineCtx, span := telemetry.Tracer.Start(machineCtx, spanName)
	defer span.End()

	if err := zeroStep.Execute(machineCtx); err != nil {
		return err
	}
	step := firstStep

	ticker := time.NewTicker(time.Duration(timeBetweenStepsInSeconds) * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-machineCtx.Done():
			return nil
		case <-ticker.C:
			// If cancellation won the race with ticker, stop before running another step.
			if machineCtx.Err() != nil {
				return nil
			}
			slog.InfoContext(machineCtx, "executing step", "step", step.Name())

			if err := step.Validate(); err != nil {
				return err
			}
			if err := step.Execute(machineCtx); err != nil {
				return err
			}

			nextStep := stateMap[step]
			if nextStep == nil {
				return nil
			} else {
				nextStep := services.PickRandomWeighted(nextStep)
				slog.InfoContext(machineCtx, "transitioning to state", "oldStep", step.Name(), "newStep", nextStep.Name())
				step = nextStep
			}
		}
	}
}
