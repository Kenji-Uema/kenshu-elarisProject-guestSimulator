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

func WaitCheckinTomorrowMessage(ctx context.Context, state *domain.State, cache port.Cache, bus *GuestCommunicationBus) error {
	ch := make(chan *communication.CheckInTomorrowNotification, 1)
	unsubscribeFn := subscribeCheckinTomorrow(bus, ch)
	defer unsubscribeFn()

	for {
		if msg, ok := takePendingCheckinTomorrow(bus); ok {
			if err := matchCheckinTomorrow(ctx, state, cache, msg); err == nil {
				slog.InfoContext(ctx, "checkin tomorrow matched from pending queue",
					"bookingId", msg.GetBookingId(),
					"guestId", msg.GetGuestId(),
					"checkIn", msg.GetCheckIn().AsTime().UTC())
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
			if err := matchCheckinTomorrow(ctx, state, cache, msg); err == nil {
				slog.InfoContext(ctx, "checkin tomorrow matched from subscriber channel",
					"bookingId", msg.GetBookingId(),
					"guestId", msg.GetGuestId(),
					"checkIn", msg.GetCheckIn().AsTime().UTC())
				return nil
			}
		}
	}
}

func matchCheckinTomorrow(ctx context.Context, state *domain.State, cache port.Cache, msg *communication.CheckInTomorrowNotification) error {
	if msg == nil {
		return fmt.Errorf("checkin tomorrow notification is nil")
	}
	cacheValue, err := cache.Load(ctx, state)
	if err != nil {
		return err
	}
	if cacheValue.Booking == nil || cacheValue.Booking.SelectedPeriod == nil {
		return fmt.Errorf("cached booking context is empty")
	}
	if msg.GetBookingId() != cacheValue.Booking.BookingID {
		return fmt.Errorf("checkin tomorrow bookingId %q does not match cache %q", msg.GetBookingId(), cacheValue.Booking.BookingID)
	}
	if msg.GetGuestId() != state.GuestId {
		return fmt.Errorf("checkin tomorrow guestId %q does not match state %q", msg.GetGuestId(), state.GuestId)
	}
	if msg.GetCottageName() != cacheValue.Booking.SelectedCottage {
		return fmt.Errorf("checkin tomorrow cottage %q does not match selected cottage %q", msg.GetCottageName(), cacheValue.Booking.SelectedCottage)
	}
	if msg.GetCheckIn() == nil || !msg.GetCheckIn().AsTime().UTC().Equal(cacheValue.Booking.SelectedPeriod.Start.UTC()) {
		return fmt.Errorf("checkin tomorrow checkIn does not match selected period start %s", cacheValue.Booking.SelectedPeriod.Start.UTC())
	}
	return nil
}

func subscribeCheckinTomorrow(bus *GuestCommunicationBus, ch chan<- *communication.CheckInTomorrowNotification) func() {
	bus.checkinTomorrowMu.Lock()
	bus.checkinTomorrowChannels.Add(ch)
	bus.checkinTomorrowMu.Unlock()

	return func() {
		bus.checkinTomorrowMu.Lock()
		bus.checkinTomorrowChannels.Remove(ch)
		bus.checkinTomorrowMu.Unlock()
	}
}

func takePendingCheckinTomorrow(bus *GuestCommunicationBus) (*communication.CheckInTomorrowNotification, bool) {
	bus.checkinTomorrowMu.Lock()
	defer bus.checkinTomorrowMu.Unlock()
	if len(bus.pendingCheckinTomorrow) == 0 {
		return nil, false
	}

	msg := bus.pendingCheckinTomorrow[0]
	bus.pendingCheckinTomorrow = bus.pendingCheckinTomorrow[1:]
	return msg, true
}
