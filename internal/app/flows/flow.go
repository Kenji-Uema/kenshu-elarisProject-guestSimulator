package flows

import (
	"context"
	"log/slog"
	"time"

	"github.com/Kenji-Uema/guestSimulator/internal/app/services"
	"github.com/Kenji-Uema/guestSimulator/internal/app/steps"
	"github.com/Kenji-Uema/guestSimulator/internal/domain"
	"github.com/Kenji-Uema/guestSimulator/internal/infra/telemetry"
)

type Flow struct {
	spanName                  string
	zeroStep                  steps.Step
	firstStep                 steps.Step
	stateMap                  map[steps.Step][]domain.WeightedTuple[steps.Step]
	timeBetweenStepsInSeconds int
}

type Startable interface {
	Start(ctx context.Context) error
}

func (f *Flow) Start(ctx context.Context) error {
	return runStepGraph(ctx, f.spanName, f.zeroStep, f.firstStep, f.stateMap, f.timeBetweenStepsInSeconds)
}

func runStepGraph(ctx context.Context, spanName string, zeroStep steps.Step, firstStep steps.Step,
	stateMap map[steps.Step][]domain.WeightedTuple[steps.Step], timeBetweenStepsInSeconds int) error {
	flowCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	flowCtx, span := telemetry.Tracer.Start(flowCtx, spanName)
	defer span.End()

	if err := zeroStep.Execute(flowCtx); err != nil {
		return err
	}
	step := firstStep

	ticker := time.NewTicker(time.Duration(timeBetweenStepsInSeconds) * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-flowCtx.Done():
			return nil
		case <-ticker.C:
			// If cancellation won the race with ticker, stop before running another step.
			if flowCtx.Err() != nil {
				return nil
			}
			slog.InfoContext(flowCtx, "executing step", "step", step.Name())

			if err := step.Validate(); err != nil {
				return err
			}
			if err := step.Execute(flowCtx); err != nil {
				return err
			}

			nextStep := stateMap[step]
			if nextStep == nil {
				return nil
			} else {
				nextStep := services.PickRandomWeighted(nextStep)
				slog.InfoContext(flowCtx, "transitioning to state", "oldStep", step.Name(), "newStep", nextStep.Name())
				step = nextStep
			}
		}
	}
}
