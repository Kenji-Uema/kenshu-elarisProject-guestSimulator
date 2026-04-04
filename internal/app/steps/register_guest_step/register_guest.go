package register_guest_step

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/Kenji-Uema/guestSimulator/internal/app/journeyctx"
	"github.com/Kenji-Uema/guestSimulator/internal/domain"
	"github.com/Kenji-Uema/guestSimulator/internal/domain/dto"
	redisc "github.com/Kenji-Uema/guestSimulator/internal/infra/redis"
	"github.com/Kenji-Uema/guestSimulator/internal/infra/telemetry"
	"github.com/go-resty/resty/v2"
	"go.opentelemetry.io/otel/attribute"
)

type RegisterGuestStep struct {
	client *resty.Client
	redis  *redisc.Redis
	state  *domain.State
}

func NewRegisterGuestStep(c *resty.Client, redis *redisc.Redis, state *domain.State) *RegisterGuestStep {
	return &RegisterGuestStep{client: c, redis: redis, state: state}
}

func (s RegisterGuestStep) Name() string {
	return "RegisterGuestStep"
}

func (s RegisterGuestStep) Validate() error {
	if s.state == nil {
		return fmt.Errorf("invalid state, state is nil")
	}
	if s.redis == nil {
		return fmt.Errorf("invalid redis client")
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
	if _, err := journeyctx.EnsureRedisKey(s.state); err != nil {
		return err
	}
	if err := journeyctx.Save(ctx, s.redis, s.state, dto.GuestJourneyCacheValue{
		GuestID:      guestId,
		PersonalInfo: guest,
	}); err != nil {
		return err
	}
	slog.InfoContext(ctx, "state updated with guestId", "guestId", guestId)

	return nil
}
