package journey_services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/Kenji-Uema/guestSimulator/internal/domain"
	"github.com/Kenji-Uema/guestSimulator/internal/domain/dto/communication"
	"github.com/Kenji-Uema/guestSimulator/internal/port"
)

func WaitBookingConfirmationMessage(ctx context.Context, state *domain.State, cache port.Cache, bus *GuestCommunicationBus) error {
	ch := make(chan *communication.BookingConfirmedNotificationEvent, 1)
	unsubscribeFn := subscribeBookingConfirmation(bus, ch)
	defer unsubscribeFn()

	for {
		if msg, ok := takePendingBookingConfirmation(bus); ok {
			if err := matchBookingConfirmation(ctx, state, cache, msg); err == nil {
				slog.InfoContext(ctx, "booking confirmation matched from pending queue",
					"bookingId", msg.GetBookingId(),
					"guestId", msg.GetGuest().GetGuestId())
				return nil
			}
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-bus.Done():
			if err := bus.Err(); err != nil && !errors.Is(err, context.Canceled) {
				return err
			}
			return fmt.Errorf("guest communication channel closed")
		case msg := <-ch:
			if err := matchBookingConfirmation(ctx, state, cache, msg); err == nil {
				slog.InfoContext(ctx, "booking confirmation matched from subscriber channel",
					"bookingId", msg.GetBookingId(),
					"guestId", msg.GetGuest().GetGuestId())
				return nil
			}
		}
	}
}

func matchBookingConfirmation(ctx context.Context, state *domain.State, cache port.Cache, msg *communication.BookingConfirmedNotificationEvent) error {
	if msg == nil {
		return fmt.Errorf("booking confirmation is nil")
	}
	cacheValue, err := cache.Load(ctx, state)
	if err != nil {
		return err
	}
	if cacheValue.Booking == nil {
		return fmt.Errorf("cached booking context is empty")
	}
	if msg.GetBookingId() != cacheValue.Booking.BookingID {
		return fmt.Errorf("booking confirmation bookingId %q does not match cache %q", msg.GetBookingId(), cacheValue.Booking.BookingID)
	}
	if msg.GetBookingStatus() != communication.BookingStatus_BOOKING_STATUS_CONFIRMED {
		return fmt.Errorf("booking confirmation status %q is not confirmed", msg.GetBookingStatus().String())
	}
	if msg.GetGuest() == nil || msg.GetGuest().GetGuestId() != state.GuestId {
		return fmt.Errorf("booking confirmation guestId does not match guest")
	}
	return nil
}

func subscribeBookingConfirmation(bus *GuestCommunicationBus, ch chan<- *communication.BookingConfirmedNotificationEvent) func() {
	bus.bookingConfirmationMu.Lock()
	bus.bookingConfirmationChannels.Add(ch)
	bus.bookingConfirmationMu.Unlock()

	return func() {
		bus.bookingConfirmationMu.Lock()
		bus.bookingConfirmationChannels.Remove(ch)
		bus.bookingConfirmationMu.Unlock()
	}
}

func takePendingBookingConfirmation(bus *GuestCommunicationBus) (*communication.BookingConfirmedNotificationEvent, bool) {
	bus.bookingConfirmationMu.Lock()
	defer bus.bookingConfirmationMu.Unlock()
	if len(bus.pendingBookingConfirmations) == 0 {
		return nil, false
	}

	msg := bus.pendingBookingConfirmations[0]
	bus.pendingBookingConfirmations = bus.pendingBookingConfirmations[1:]
	return msg, true
}
