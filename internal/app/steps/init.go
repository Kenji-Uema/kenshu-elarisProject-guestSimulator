package steps

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/Kenji-Uema/guestSimulator/internal/domain"
	"github.com/Kenji-Uema/guestSimulator/internal/tooling/telemetry"
	"github.com/brianvoe/gofakeit/v7"
	"go.opentelemetry.io/otel/attribute"
)

type InitStep struct {
	state *domain.State
}

func NewInitStep(state *domain.State) Step {
	return &InitStep{state: state}
}

func (s InitStep) Name() string {
	return "InitStep"
}

func (s InitStep) Validate() error {
	if s.state == nil {
		return fmt.Errorf("invalid state, state is nil")
	}
	return nil
}

func (s InitStep) Execute(ctx context.Context) error {
	ctx, span := telemetry.Tracer.Start(ctx, "InitStep")
	defer span.End()

	slog.InfoContext(ctx, "Start process of booking a cottage")

	name := strings.Split(gofakeit.Name(), " ")

	guest := domain.Guest{
		DocumentId: gofakeit.SSN(),
		GivenNames: name[0],
		Surname:    name[1],
		Email:      fmt.Sprintf("%s.%s@test.com", name[0], name[1]),
	}

	span.SetAttributes(attribute.String("guest.Email", guest.Email))

	s.state.Guest = &guest
	slog.InfoContext(ctx, "state updated with new Guest", "guest", guest)

	return nil
}
