package booking_step

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"

	"github.com/Kenji-Uema/guestSimulator/internal/app/steps"
	"github.com/Kenji-Uema/guestSimulator/internal/domain"
	"github.com/Kenji-Uema/guestSimulator/internal/domain/dto"
	"github.com/Kenji-Uema/guestSimulator/internal/infra/telemetry"
	"github.com/Kenji-Uema/guestSimulator/internal/port"
)

type SelectCottageStep struct {
	state *domain.State
	cache port.Cache
}

func NewSelectCottageStep(state *domain.State, cache port.Cache) steps.Step {
	return &SelectCottageStep{state: state, cache: cache}
}

func (s SelectCottageStep) Name() string {
	return "SelectCottageStep"
}

func (s SelectCottageStep) Validate() error {
	if s.state == nil {
		return fmt.Errorf("invalid state, state is nil")
	}
	if s.cache == nil {
		return fmt.Errorf("invalid guest journey cache")
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
	cacheValue, err := s.cache.Load(ctx, s.state)
	if err != nil {
		return err
	}
	if cacheValue.Booking == nil {
		cacheValue.Booking = &dto.GuestJourneyBooking{}
	}
	cacheValue.Booking.SelectedCottage = selectedCottage
	if err := s.cache.Save(ctx, s.state, cacheValue); err != nil {
		return err
	}
	slog.InfoContext(ctx, "state updated, cottage selected", "selectedCottage", selectedCottage)

	return nil
}
