package journey_services

import (
	"context"
	"testing"
	"time"

	"github.com/Kenji-Uema/guestSimulator/internal/domain"
	"github.com/Kenji-Uema/guestSimulator/internal/domain/dto"
	"github.com/Kenji-Uema/guestSimulator/internal/domain/dto/booking"
	"github.com/Kenji-Uema/guestSimulator/internal/domain/dto/communication"
	redisfakes "github.com/Kenji-Uema/guestSimulator/internal/infra/redis/fakes"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestSubscribeAndTakePendingCheckinToday(t *testing.T) {
	bus := NewGuestCommunicationBus()
	ch := make(chan *communication.CheckInTodayNotification, 1)
	unsubscribe := subscribeCheckinToday(bus, ch)

	if len(bus.checkinTodayChannels.Values()) != 1 {
		t.Fatalf("unexpected subscriber count: %d", len(bus.checkinTodayChannels.Values()))
	}

	unsubscribe()

	if len(bus.checkinTodayChannels.Values()) != 0 {
		t.Fatalf("unexpected subscriber count after unsubscribe: %d", len(bus.checkinTodayChannels.Values()))
	}

	expected := &communication.CheckInTodayNotification{BookingId: "booking-1"}
	bus.pendingCheckinToday = []*communication.CheckInTodayNotification{expected}

	got, ok := takePendingCheckinToday(bus)
	if !ok || got != expected {
		t.Fatalf("unexpected pending checkin notification: %#v %v", got, ok)
	}
}

func TestMatchCheckinTodaySuccess(t *testing.T) {
	checkIn := time.Date(2026, time.April, 12, 0, 0, 0, 0, time.UTC)
	cache := &redisfakes.Cache{
		LoadValue: dto.GuestJourneyCacheValue{
			Booking: &dto.GuestJourneyBooking{
				BookingID:       "booking-1",
				SelectedCottage: "Alps",
				SelectedPeriod:  &booking.Period{Start: checkIn, End: checkIn.AddDate(0, 0, 3)},
			},
		},
	}

	err := matchCheckinToday(context.Background(), &domain.State{GuestId: "guest-1"}, cache, &communication.CheckInTodayNotification{
		BookingId:   "booking-1",
		GuestId:     "guest-1",
		CottageName: "Alps",
		CheckIn:     timestamppb.New(checkIn),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
