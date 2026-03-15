package register_guest_step

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/Kenji-Uema/guestSimulator/internal/domain"
	"github.com/Kenji-Uema/guestSimulator/internal/tooling/telemetry"
	"github.com/go-resty/resty/v2"
	"go.opentelemetry.io/otel/attribute"
)

type RegisterGuestStep struct {
	client *resty.Client
	state  *domain.State
}

func NewRegisterGuestStep(c *resty.Client, state *domain.State) *RegisterGuestStep {
	return &RegisterGuestStep{client: c, state: state}
}

func (s RegisterGuestStep) Name() string {
	return "RegisterGuestStep"
}

func (s RegisterGuestStep) Validate() error {
	if s.state == nil {
		return fmt.Errorf("invalid state, state is nil")
	}

	if s.state.Guest == nil {
		return fmt.Errorf("invalid state, guest is nil")
	}

	return nil
}

func (s RegisterGuestStep) Execute(ctx context.Context) error {
	ctx, span := telemetry.Tracer.Start(ctx, "RegisterGuestStep")
	defer span.End()

	guest := s.state.Guest
	slog.InfoContext(ctx, "User registers its own account", "guest.Email", guest.Email)

	resp, err := s.client.R().
		SetContext(ctx).
		SetBody(guest).
		Post("/guest")

	if err != nil {
		return err
	}

	if resp.IsError() {
		return fmt.Errorf("error: %s", resp.Error())
	}

	var guestId string
	if err := json.Unmarshal(resp.Body(), &guestId); err != nil {
		return err
	}

	span.SetAttributes(attribute.String("guest.ID", guestId))

	s.state.GuestId = guestId
	slog.InfoContext(ctx, "state updated with guestId", "guestId", guestId)

	return nil
}
