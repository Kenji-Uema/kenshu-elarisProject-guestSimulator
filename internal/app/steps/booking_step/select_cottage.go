package booking_step

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"

	"github.com/Kenji-Uema/guestEmulator/internal/app/steps"
	"github.com/Kenji-Uema/guestEmulator/internal/domain"
	"github.com/Kenji-Uema/guestEmulator/internal/tooling/telemetry"
)

type SelectCottageStep struct {
	state *domain.State
}

func NewSelectCottageStep(state *domain.State) steps.Step {
	return &SelectCottageStep{state: state}
}

func (s SelectCottageStep) Name() string {
	return "SelectCottageStep"
}

func (s SelectCottageStep) Validate() error {
	if s.state == nil {
		return fmt.Errorf("invalid state, state is nil")
	}

	if s.state.CottageNames != nil && len(s.state.CottageNames) == 0 {
		return fmt.Errorf("invalid state, cottageNames is empty")
	}

	return nil
}

func (s SelectCottageStep) Execute(ctx context.Context) error {
	ctx, span := telemetry.Tracer.Start(ctx, "SelectCottageStep")
	defer span.End()

	s.state.SelectedCottage = s.state.CottageNames[rand.Intn(len(s.state.CottageNames))]
	slog.InfoContext(ctx, "state updated, cottage selected", "selectedCottage", s.state.SelectedCottage)

	return nil
}
