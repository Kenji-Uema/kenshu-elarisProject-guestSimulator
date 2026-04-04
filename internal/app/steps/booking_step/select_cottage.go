package booking_step

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"

	"github.com/Kenji-Uema/guestSimulator/internal/app/journeyctx"
	"github.com/Kenji-Uema/guestSimulator/internal/app/steps"
	"github.com/Kenji-Uema/guestSimulator/internal/domain"
	"github.com/Kenji-Uema/guestSimulator/internal/domain/dto"
	redisc "github.com/Kenji-Uema/guestSimulator/internal/infra/redis"
	"github.com/Kenji-Uema/guestSimulator/internal/infra/telemetry"
)

type SelectCottageStep struct {
	state *domain.State
	redis *redisc.Redis
}

func NewSelectCottageStep(state *domain.State, redis *redisc.Redis) steps.Step {
	return &SelectCottageStep{state: state, redis: redis}
}

func (s SelectCottageStep) Name() string {
	return "SelectCottageStep"
}

func (s SelectCottageStep) Validate() error {
	if s.state == nil {
		return fmt.Errorf("invalid state, state is nil")
	}
	if s.redis == nil {
		return fmt.Errorf("invalid redis client")
	}

	if s.state.CottageNames != nil && len(s.state.CottageNames) == 0 {
		return fmt.Errorf("invalid state, cottageNames is empty")
	}

	return nil
}

func (s SelectCottageStep) Execute(ctx context.Context) error {
	ctx, span := telemetry.Tracer.Start(ctx, "SelectCottageStep")
	defer span.End()

	selectedCottage := s.state.CottageNames[rand.Intn(len(s.state.CottageNames))]
	cacheValue, err := journeyctx.Load(ctx, s.redis, s.state)
	if err != nil {
		return err
	}
	if cacheValue.Booking == nil {
		cacheValue.Booking = &dto.GuestJourneyBooking{}
	}
	cacheValue.Booking.SelectedCottage = selectedCottage
	if err := journeyctx.Save(ctx, s.redis, s.state, cacheValue); err != nil {
		return err
	}
	slog.InfoContext(ctx, "state updated, cottage selected", "selectedCottage", selectedCottage)

	return nil
}
