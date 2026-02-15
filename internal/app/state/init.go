package state

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/Kenji-Uema/guestEmulator/internal/domain"
	"github.com/Kenji-Uema/guestEmulator/internal/tooling/telemetry"
	"github.com/brianvoe/gofakeit/v7"
	"go.opentelemetry.io/otel/attribute"
)

type InitState struct{}

func NewInitState() *InitState {
	return &InitState{}
}

func (s InitState) Execute(ctx context.Context) (context.Context, error) {
	ctx, span := telemetry.Tracer.Start(ctx, "InitState")
	defer span.End()

	slog.InfoContext(ctx, "Start process of booking a cottage")

	name := strings.Split(gofakeit.Name(), " ")

	guest := domain.Guest{
		DocumentId: gofakeit.SSN(),
		GivenNames: name[0],
		Surname:    name[1],
		Email:      fmt.Sprintf("%s.%s@test.com", name[0], name[1]),
	}

	slog.InfoContext(ctx, "New person generated", "guest", guest)
	span.SetAttributes(attribute.String("guest.Email", guest.Email))

	return context.WithValue(ctx, "guest", guest), nil
}
