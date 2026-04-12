package journey_services

import (
	"context"
	"testing"

	"github.com/Kenji-Uema/guestSimulator/internal/domain"
	"github.com/Kenji-Uema/guestSimulator/internal/domain/dto"
	"github.com/Kenji-Uema/guestSimulator/internal/domain/dto/communication"
	redisfakes "github.com/Kenji-Uema/guestSimulator/internal/infra/redis/fakes"
)

func TestSubscribeAndTakePendingBookingConfirmation(t *testing.T) {
	bus := NewGuestCommunicationBus()
	ch := make(chan *communication.BookingConfirmedNotificationEvent, 1)
	unsubscribe := subscribeBookingConfirmation(bus, ch)

	if len(bus.bookingConfirmationChannels.Values()) != 1 {
		t.Fatalf("unexpected subscriber count: %d", len(bus.bookingConfirmationChannels.Values()))
	}

	unsubscribe()

	if len(bus.bookingConfirmationChannels.Values()) != 0 {
		t.Fatalf("unexpected subscriber count after unsubscribe: %d", len(bus.bookingConfirmationChannels.Values()))
	}

	expected := &communication.BookingConfirmedNotificationEvent{BookingId: "booking-1"}
	bus.pendingBookingConfirmations = []*communication.BookingConfirmedNotificationEvent{expected}

	got, ok := takePendingBookingConfirmation(bus)
	if !ok || got != expected {
		t.Fatalf("unexpected pending booking confirmation: %#v %v", got, ok)
	}
}

func TestMatchBookingConfirmationSuccess(t *testing.T) {
	cache := &redisfakes.Cache{
		LoadValue: dto.GuestJourneyCacheValue{
			Booking: &dto.GuestJourneyBooking{BookingID: "booking-1"},
		},
	}

	err := matchBookingConfirmation(context.Background(), &domain.State{GuestId: "guest-1"}, cache, &communication.BookingConfirmedNotificationEvent{
		BookingId:     "booking-1",
		BookingStatus: communication.BookingStatus_BOOKING_STATUS_CONFIRMED,
		Guest:         &communication.Guest{GuestId: "guest-1"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
