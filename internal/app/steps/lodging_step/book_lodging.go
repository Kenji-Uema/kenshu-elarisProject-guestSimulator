package lodging_step

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/Kenji-Uema/guestSimulator/internal/app/steps"
	"github.com/Kenji-Uema/guestSimulator/internal/domain"
	"github.com/Kenji-Uema/guestSimulator/internal/domain/dto/booking"
	"github.com/Kenji-Uema/guestSimulator/internal/infra/telemetry"
	"github.com/Kenji-Uema/guestSimulator/internal/port"
	"github.com/go-resty/resty/v2"
)

type BookLodgingStep struct {
	client *resty.Client
	state  *domain.State
	cache  port.Cache
}

func NewBookLodgingStep(state *domain.State, client *resty.Client, cache port.Cache) steps.Step {
	return &BookLodgingStep{client: client, state: state, cache: cache}
}

func (s BookLodgingStep) Name() string {
	return "BookLodgingStep"
}

func (s BookLodgingStep) Validate() error {
	if s.state == nil {
		return fmt.Errorf("invalid state, state is nil")
	}
	if s.state.GuestId == "" {
		return fmt.Errorf("invalid state, guestId is empty")
	}
	if s.cache == nil {
		return fmt.Errorf("invalid guest journey cache")
	}

	return nil
}

func (s BookLodgingStep) Execute(ctx context.Context) error {
	ctx, span := telemetry.Tracer.Start(ctx, "BookLodgingStep")
	defer span.End()

	cacheValue, err := s.cache.Load(ctx, s.state)
	if err != nil {
		return err
	}
	if cacheValue.Booking == nil || cacheValue.Booking.SelectedCottage == "" || cacheValue.Booking.SelectedPeriod == nil {
		return fmt.Errorf("invalid cached bookingConfirmation context")
	}

	resp, err := s.client.R().
		SetContext(ctx).
		SetBody(booking.BookingRequest{
			GuestId:        s.state.GuestId,
			NumberOfGuests: 1,
			CheckInDate:    cacheValue.Booking.SelectedPeriod.Start.Format("2006-01-02"),
			CheckOutDate:   cacheValue.Booking.SelectedPeriod.End.Format("2006-01-02"),
		}).
		Post(fmt.Sprintf("/cottage/%s/bookingConfirmation", cacheValue.Booking.SelectedCottage))
	if err != nil {
		return err
	}
	if resp.IsError() {
		return fmt.Errorf("error: %s", resp.Status())
	}

	var bookingConfirmation booking.BookingConfirmation
	if err := json.Unmarshal(resp.Body(), &bookingConfirmation); err != nil {
		return err
	}

	cacheValue.Booking.BookingID = bookingConfirmation.Id
	if err := s.cache.Save(ctx, s.state, cacheValue); err != nil {
		return err
	}
	slog.InfoContext(ctx, "state updated with lodging bookingConfirmation id", "bookingId", bookingConfirmation.Id)

	return nil
}
