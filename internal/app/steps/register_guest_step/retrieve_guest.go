package register_guest_step

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Kenji-Uema/guestEmulator/internal/domain"
	"github.com/Kenji-Uema/guestEmulator/internal/tooling/telemetry"
	"github.com/go-resty/resty/v2"
)

type RetrieveGuestStep struct {
	client *resty.Client
	state  *domain.State
}

func NewRetrieveGuestStep(c *resty.Client, state *domain.State) *RetrieveGuestStep {
	return &RetrieveGuestStep{client: c, state: state}
}

func (s RetrieveGuestStep) Name() string {
	return "RetrieveGuestStep"
}

func (s RetrieveGuestStep) Validate() error {
	if s.state.GuestId == "" {
		return fmt.Errorf("invalid state, guestId is empty")
	}

	return nil
}

func (s RetrieveGuestStep) Execute(ctx context.Context) error {
	ctx, span := telemetry.Tracer.Start(ctx, "RetrieveGuestStep")
	defer span.End()

	slog.InfoContext(ctx, "User retrieves its own account")

	resp, err := s.client.R().
		SetContext(ctx).
		Get(fmt.Sprintf("/guest/%s", s.state.GuestId))

	if err != nil {
		return err
	}

	if resp.IsError() {
		return fmt.Errorf("error: %s", resp.Status())
	}

	slog.InfoContext(ctx, "Guest retrieved correctly", "guest", string(resp.Body()))

	return nil
}
