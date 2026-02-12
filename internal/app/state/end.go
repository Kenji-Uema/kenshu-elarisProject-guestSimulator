package state

import (
	"context"
	"log/slog"

	"github.com/Kenji-Uema/guestEmulator/internal/domain"
)

type EndState struct{}

func NewEndState() *EndState {
	return &EndState{}
}

func (e EndState) Execute(ctx context.Context, _ domain.IgnoredField) (domain.IgnoredField, error) {
	slog.Info("Guest ended interaction with the system")

	ctx.Done()

	return domain.IgnoredField{}, nil
}
