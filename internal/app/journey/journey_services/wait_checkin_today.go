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

func WaitCheckinTodayMessage(ctx context.Context, state *domain.State, cache port.Cache, bus *GuestCommunicationBus) error {
	ch := make(chan *communication.CheckInTodayNotification, 1)
	unsubscribeFn := subscribeCheckinToday(bus, ch)
	defer unsubscribeFn()

	for {
		if msg, ok := takePendingCheckinToday(bus); ok {
			if err := matchCheckinToday(ctx, state, cache, msg); err == nil {
				slog.InfoContext(ctx, "checkin today matched from pending queue",
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
			if err := matchCheckinToday(ctx, state, cache, msg); err == nil {
				slog.InfoContext(ctx, "checkin today matched from subscriber channel",
					"bookingId", msg.GetBookingId(),
					"guestId", msg.GetGuestId(),
					"checkIn", msg.GetCheckIn().AsTime().UTC())
				return nil
			}
		}
	}
}

func matchCheckinToday(ctx context.Context, state *domain.State, cache port.Cache, msg *communication.CheckInTodayNotification) error {
	if msg == nil {
		return fmt.Errorf("checkin today notification is nil")
	}
	cacheValue, err := cache.Load(ctx, state)
	if err != nil {
		return err
	}
	if cacheValue.Booking == nil || cacheValue.Booking.SelectedPeriod == nil {
		return fmt.Errorf("cached booking context is empty")
	}
	if msg.GetBookingId() != cacheValue.Booking.BookingID {
		return fmt.Errorf("checkin today bookingId %q does not match cache %q", msg.GetBookingId(), cacheValue.Booking.BookingID)
	}
	if msg.GetGuestId() != state.GuestId {
		return fmt.Errorf("checkin today guestId %q does not match state %q", msg.GetGuestId(), state.GuestId)
	}
	if msg.GetCottageName() != cacheValue.Booking.SelectedCottage {
		return fmt.Errorf("checkin today cottage %q does not match selected cottage %q", msg.GetCottageName(), cacheValue.Booking.SelectedCottage)
	}
	if msg.GetCheckIn() == nil || !msg.GetCheckIn().AsTime().UTC().Equal(cacheValue.Booking.SelectedPeriod.Start.UTC()) {
		return fmt.Errorf("checkin today checkIn does not match selected period start %s", cacheValue.Booking.SelectedPeriod.Start.UTC())
	}
	return nil
}

func subscribeCheckinToday(bus *GuestCommunicationBus, ch chan<- *communication.CheckInTodayNotification) func() {
	bus.checkinTodayMu.Lock()
	bus.checkinTodayChannels.Add(ch)
	bus.checkinTodayMu.Unlock()

	return func() {
		bus.checkinTodayMu.Lock()
		bus.checkinTodayChannels.Remove(ch)
		bus.checkinTodayMu.Unlock()
	}
}

func takePendingCheckinToday(bus *GuestCommunicationBus) (*communication.CheckInTodayNotification, bool) {
	bus.checkinTodayMu.Lock()
	defer bus.checkinTodayMu.Unlock()
	if len(bus.pendingCheckinToday) == 0 {
		return nil, false
	}

	msg := bus.pendingCheckinToday[0]
	bus.pendingCheckinToday = bus.pendingCheckinToday[1:]
	return msg, true
}
