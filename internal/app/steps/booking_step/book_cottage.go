package booking_step

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/Kenji-Uema/guestSimulator/internal/app/steps"
	"github.com/Kenji-Uema/guestSimulator/internal/domain"
	"github.com/Kenji-Uema/guestSimulator/internal/tooling/telemetry"

	"github.com/go-resty/resty/v2"
)

type BookCottageStep struct {
	client *resty.Client
	state  *domain.State
}

func NewBookCottageStep(state *domain.State, c *resty.Client) steps.Step {
	return &BookCottageStep{client: c, state: state}
}

func (s BookCottageStep) Name() string {
	return "BookCottageStep"
}

func (s BookCottageStep) Validate() error {
	if s.state == nil {
		return fmt.Errorf("invalid state, state is nil")
	}

	if s.state.GuestId == "" {
		return fmt.Errorf("invalid state, guestId is empty")
	}

	if s.state.SelectedCottage == "" {
		return fmt.Errorf("invalid state, selectedCottage is empty")
	}

	if s.state.SelectedPeriod == nil {
		return fmt.Errorf("invalid state, selectedPeriod is empty")
	}

	return nil
}

func (s BookCottageStep) Execute(ctx context.Context) error {
	ctx, span := telemetry.Tracer.Start(ctx, "BookCottageStep")
	defer span.End()

	slog.InfoContext(ctx, "User book a cottage", "cottage", s.state.Guest.Email)

	resp, err := s.client.R().
		SetContext(ctx).
		SetBody(domain.BookingRequest{
			GuestId:        s.state.GuestId,
			NumberOfGuests: 1,
			CheckInDate:    s.state.SelectedPeriod.Start.Format("2006-01-02"),
			CheckOutDate:   s.state.SelectedPeriod.End.Format("2006-01-02")}).
		Post(fmt.Sprintf("/cottage/%s/booking", s.state.SelectedCottage))

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
	s.state.BookingId = bookingConfirmation.Id
	slog.InfoContext(ctx, "state update, added bookingId", "bookingId", bookingConfirmation.Id)

	return nil
}
