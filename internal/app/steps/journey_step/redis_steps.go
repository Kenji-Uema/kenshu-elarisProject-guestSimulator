package journey_step

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Kenji-Uema/guestSimulator/internal/app/steps"
	"github.com/Kenji-Uema/guestSimulator/internal/domain"
	"github.com/Kenji-Uema/guestSimulator/internal/domain/dto"
	"github.com/Kenji-Uema/guestSimulator/internal/infra/telemetry"
	"github.com/Kenji-Uema/guestSimulator/internal/port"
	"go.opentelemetry.io/otel/attribute"
)

type upsertGuestCacheStep struct {
	name  string
	state *domain.State
	cache port.Cache
}

type logGuestCacheStep struct {
	state *domain.State
	cache port.Cache
}

type deleteGuestCacheStep struct {
	state *domain.State
	cache port.Cache
}

func NewSaveGuestCacheStep(state *domain.State, cache port.Cache) steps.Step {
	return &upsertGuestCacheStep{name: "SaveGuestCacheStep", state: state, cache: cache}
}

func NewUpdateBookingCacheStep(state *domain.State, cache port.Cache) steps.Step {
	return &upsertGuestCacheStep{name: "UpdateBookingCacheStep", state: state, cache: cache}
}

func NewUpdateInvoiceCacheStep(state *domain.State, cache port.Cache) steps.Step {
	return &upsertGuestCacheStep{name: "UpdateInvoiceCacheStep", state: state, cache: cache}
}

func NewLogGuestCacheStep(state *domain.State, cache port.Cache) steps.Step {
	return &logGuestCacheStep{state: state, cache: cache}
}

func NewDeleteGuestCacheStep(state *domain.State, cache port.Cache) steps.Step {
	return &deleteGuestCacheStep{state: state, cache: cache}
}

func (s upsertGuestCacheStep) Name() string {
	return s.name
}

func (s upsertGuestCacheStep) Validate() error {
	if s.state == nil {
		return fmt.Errorf("invalid state, state is nil")
	}
	if s.cache == nil {
		return fmt.Errorf("invalid guest journey cache")
	}
	if s.state.GuestId == "" {
		return fmt.Errorf("invalid state, guestId is empty")
	}
	return nil
}

func (s upsertGuestCacheStep) Execute(ctx context.Context) error {
	ctx, span := telemetry.Tracer.Start(ctx, s.name)
	defer span.End()

	key, err := s.cache.EnsureKey(s.state)
	if err != nil {
		return err
	}

	value := buildCacheValue(s.state)
	if current, err := s.cache.Load(ctx, s.state); err == nil {
		value = mergeCacheValue(current, value)
	}
	if err := s.cache.Save(ctx, s.state, value); err != nil {
		return err
	}

	slog.InfoContext(ctx, "guest journey cache upserted", "key", key, "step", s.name)
	return nil
}

func (s logGuestCacheStep) Name() string {
	return "LogGuestCacheStep"
}

func (s logGuestCacheStep) Validate() error {
	if s.state == nil {
		return fmt.Errorf("invalid state, state is nil")
	}
	if s.cache == nil {
		return fmt.Errorf("invalid guest journey cache")
	}
	if s.state.RedisKey == "" {
		return fmt.Errorf("invalid state, redisKey is empty")
	}
	return nil
}

func (s logGuestCacheStep) Execute(ctx context.Context) error {
	ctx, span := telemetry.Tracer.Start(ctx, "LogGuestCacheStep")
	defer span.End()

	if _, err := s.cache.EnsureKey(s.state); err != nil {
		return err
	}

	value, err := s.cache.Get(ctx, s.state.RedisKey)
	if err != nil {
		return err
	}

	slog.InfoContext(ctx, "guest journey cache content", "key", s.state.RedisKey, "value", value)
	return nil
}

func (s deleteGuestCacheStep) Name() string {
	return "DeleteGuestCacheStep"
}

func (s deleteGuestCacheStep) Validate() error {
	if s.state == nil {
		return fmt.Errorf("invalid state, state is nil")
	}
	if s.cache == nil {
		return fmt.Errorf("invalid guest journey cache")
	}
	if s.state.RedisKey == "" {
		return fmt.Errorf("invalid state, redisKey is empty")
	}
	return nil
}

func (s deleteGuestCacheStep) Execute(ctx context.Context) error {
	ctx, span := telemetry.Tracer.Start(ctx, "DeleteGuestCacheStep")
	defer span.End()
	span.SetAttributes(attribute.String("redis.key", s.state.RedisKey))

	if err := s.cache.Del(ctx, s.state.RedisKey); err != nil {
		return err
	}

	slog.InfoContext(ctx, "guest journey cache deleted", "key", s.state.RedisKey)
	return nil
}

func mergeCacheValue(current dto.GuestJourneyCacheValue, update dto.GuestJourneyCacheValue) dto.GuestJourneyCacheValue {
	if update.GuestID != "" {
		current.GuestID = update.GuestID
	}
	if update.PersonalInfo != nil {
		current.PersonalInfo = update.PersonalInfo
	}
	if update.Booking != nil {
		if current.Booking == nil {
			current.Booking = update.Booking
		} else {
			if update.Booking.BookingID != "" {
				current.Booking.BookingID = update.Booking.BookingID
			}
			if update.Booking.SelectedCottage != "" {
				current.Booking.SelectedCottage = update.Booking.SelectedCottage
			}
			if update.Booking.SelectedPeriod != nil {
				current.Booking.SelectedPeriod = update.Booking.SelectedPeriod
			}
		}
	}
	if update.Invoice != nil {
		if current.Invoice == nil {
			current.Invoice = update.Invoice
		} else {
			if update.Invoice.InvoiceNumber != "" {
				current.Invoice.InvoiceNumber = update.Invoice.InvoiceNumber
			}
			if update.Invoice.ReceiptNumber != "" {
				current.Invoice.ReceiptNumber = update.Invoice.ReceiptNumber
			}
		}
	}

	return current
}

func buildCacheValue(state *domain.State) dto.GuestJourneyCacheValue {
	return dto.GuestJourneyCacheValue{
		GuestID:      state.GuestId,
		PersonalInfo: state.Guest,
	}
}
