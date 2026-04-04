package booking_step

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

type BookCottageStep struct {
	client *resty.Client
	state  *domain.State
	redis  *redisc.Redis
}

func NewBookCottageStep(state *domain.State, c *resty.Client, redis *redisc.Redis) steps.Step {
	return &BookCottageStep{client: c, state: state, redis: redis}
}

func (s BookCottageStep) Name() string {
	return "BookCottageStep"
}

func (s BookCottageStep) Validate() error {
	if s.state == nil {
		return fmt.Errorf("invalid state, state is nil")
	}
	if s.redis == nil {
		return fmt.Errorf("invalid redis client")
	}

	if s.state.GuestId == "" {
		return fmt.Errorf("invalid state, guestId is empty")
	}

	return nil
}

func (s BookCottageStep) Execute(ctx context.Context) error {
	ctx, span := telemetry.Tracer.Start(ctx, "BookCottageStep")
	defer span.End()

	cacheValue, err := journeyctx.Load(ctx, s.redis, s.state)
	if err != nil {
		return err
	}
	if cacheValue.PersonalInfo == nil {
		return fmt.Errorf("invalid cached guest context")
	}
	if cacheValue.Booking == nil || cacheValue.Booking.SelectedCottage == "" || cacheValue.Booking.SelectedPeriod == nil {
		return fmt.Errorf("invalid cached booking context")
	}

	guest := cacheValue.PersonalInfo
	selected := cacheValue.Booking
	slog.InfoContext(ctx, "User book a cottage", "cottage", selected.SelectedCottage)

	resp, err := s.client.R().
		SetContext(ctx).
		SetBody(domain.BookingRequest{
			GuestId:        s.state.GuestId,
			NumberOfGuests: 1,
			CheckInDate:    selected.SelectedPeriod.Start.Format("2006-01-02"),
			CheckOutDate:   selected.SelectedPeriod.End.Format("2006-01-02"),
			GuestName:      fmt.Sprintf("%s %s", guest.GivenNames, guest.Surname),
			GuestEmail:     guest.Email,
			GuestDocument:  guest.DocumentId,
			BillingAddress: guest.BillingAddress,
		}).
		Post(fmt.Sprintf("/cottage/%s/booking", selected.SelectedCottage))

	if err != nil {
		return err
	}

	if resp.IsError() {
		return fmt.Errorf("error: %s", resp.Status())
	}

	var bookingConfirmation domain.BookingConfirmation
	if err := json.Unmarshal(resp.Body(), &bookingConfirmation); err != nil {
		return err
	}
	cacheValue.Booking.BookingID = bookingConfirmation.Id
	if err := journeyctx.Save(ctx, s.redis, s.state, cacheValue); err != nil {
		return err
	}
	slog.InfoContext(ctx, "state update, added bookingId", "bookingId", bookingConfirmation.Id)

	return nil
}
