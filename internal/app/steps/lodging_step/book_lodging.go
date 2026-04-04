package lodging_step

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/Kenji-Uema/guestSimulator/internal/app/journeyctx"
	"github.com/Kenji-Uema/guestSimulator/internal/app/steps"
	"github.com/Kenji-Uema/guestSimulator/internal/domain"
	redisc "github.com/Kenji-Uema/guestSimulator/internal/infra/redis"
	"github.com/Kenji-Uema/guestSimulator/internal/infra/telemetry"
	"github.com/go-resty/resty/v2"
)

type BookLodgingStep struct {
	client *resty.Client
	state  *domain.State
	redis  *redisc.Redis
}

func NewBookLodgingStep(state *domain.State, client *resty.Client, redis *redisc.Redis) steps.Step {
	return &BookLodgingStep{client: client, state: state, redis: redis}
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
	if s.redis == nil {
		return fmt.Errorf("invalid redis client")
	}

	return nil
}

func (s BookLodgingStep) Execute(ctx context.Context) error {
	ctx, span := telemetry.Tracer.Start(ctx, "BookLodgingStep")
	defer span.End()

	cacheValue, err := journeyctx.Load(ctx, s.redis, s.state)
	if err != nil {
		return err
	}
	if cacheValue.Booking == nil || cacheValue.Booking.SelectedCottage == "" || cacheValue.Booking.SelectedPeriod == nil {
		return fmt.Errorf("invalid cached booking context")
	}

	resp, err := s.client.R().
		SetContext(ctx).
		SetBody(domain.BookingRequest{
			GuestId:        s.state.GuestId,
			NumberOfGuests: 1,
			CheckInDate:    cacheValue.Booking.SelectedPeriod.Start.Format("2006-01-02"),
			CheckOutDate:   cacheValue.Booking.SelectedPeriod.End.Format("2006-01-02"),
		}).
		Post(fmt.Sprintf("/cottage/%s/booking", cacheValue.Booking.SelectedCottage))
	if err != nil {
		return err
	}
	if resp.IsError() {
		return fmt.Errorf("error: %s", resp.Status())
	}

	var booking domain.BookingConfirmation
	if err := json.Unmarshal(resp.Body(), &booking); err != nil {
		return err
	}

	cacheValue.Booking.BookingID = booking.Id
	if err := journeyctx.Save(ctx, s.redis, s.state, cacheValue); err != nil {
		return err
	}
	slog.InfoContext(ctx, "state updated with lodging booking id", "bookingId", booking.Id)

	return nil
}
