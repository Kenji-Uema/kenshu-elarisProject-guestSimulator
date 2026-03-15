package steps

import (
	"context"
	"log/slog"

	"github.com/Kenji-Uema/guestSimulator/internal/domain"
	"github.com/Kenji-Uema/guestSimulator/internal/tooling/telemetry"
)

type EndStep struct {
	state *domain.State
}

func NewEndStep(state *domain.State) Step {
	return &EndStep{state: state}
}

func (s EndStep) Name() string {
	return "EndStep"
}

func (s EndStep) Validate() error {
	return nil
}

func (s EndStep) Execute(ctx context.Context) error {
	ctx, span := telemetry.Tracer.Start(ctx, "EndStep")
	defer span.End()

	slog.InfoContext(ctx, "Guest ended interaction with the system", "state", s.state)

	return nil
}
