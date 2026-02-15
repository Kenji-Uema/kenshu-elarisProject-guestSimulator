package state

import (
	"context"
	"log/slog"

	"github.com/Kenji-Uema/guestEmulator/internal/domain"
	"github.com/Kenji-Uema/guestEmulator/internal/tooling/telemetry"
)

type EndState struct{}

func NewEndState() *EndState {
	return &EndState{}
}

func (e EndState) Execute(ctx context.Context, _ domain.IgnoredField) (domain.IgnoredField, error) {
	ctx, span := telemetry.Tracer.Start(ctx, "EndState")
	defer span.End()

	slog.InfoContext(ctx, "Guest ended interaction with the system")

	return domain.IgnoredField{}, nil
}
