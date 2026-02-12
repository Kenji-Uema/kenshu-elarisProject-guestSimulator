package state

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/Kenji-Uema/guestEmulator/internal/domain"

	"github.com/brianvoe/gofakeit/v7"
)

type InitState struct{}

func NewInitState() *InitState {
	return &InitState{}
}

func (s InitState) Execute(ctx context.Context) (context.Context, error) {
	slog.Info("Start process of booking a cottage")

	name := strings.Split(gofakeit.Name(), " ")

	guest := domain.Guest{
		DocumentId: gofakeit.SSN(),
		GivenNames: name[0],
		Surname:    name[1],
		Email:      fmt.Sprintf("%s.%s@test.com", name[0], name[1]),
	}

	slog.Info("New person generated", "guest", guest)

	return context.WithValue(ctx, "guest", guest), nil
}
