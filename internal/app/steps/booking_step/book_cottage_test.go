package booking_step

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/Kenji-Uema/guestSimulator/internal/domain"
	"github.com/Kenji-Uema/guestSimulator/internal/domain/dto"
	bookingdto "github.com/Kenji-Uema/guestSimulator/internal/domain/dto/booking"
	"github.com/Kenji-Uema/guestSimulator/internal/domain/dto/guest_registration"
	redisfakes "github.com/Kenji-Uema/guestSimulator/internal/infra/redis/fakes"
	"github.com/go-resty/resty/v2"
)

func TestNewBookCottageStepReturnsNamedStep(t *testing.T) {
	step := NewBookCottageStep(&domain.State{}, resty.New(), nil)

	if step == nil {
		t.Fatal("expected step")
	}
	if step.Name() != "BookCottageStep" {
		t.Fatalf("unexpected step name: %q", step.Name())
	}
}

func TestBookCottageStepExecuteRejectsExistingBooking(t *testing.T) {
	cache := &redisfakes.Cache{
		LoadValue: dto.GuestJourneyCacheValue{
			PersonalInfo: &guest_registration.Guest{
				GivenNames:     "Guest",
				Surname:        "Test",
				Email:          "guest@test.com",
				DocumentId:     "123",
				BillingAddress: "Main Street",
			},
			Booking: &dto.GuestJourneyBooking{
				BookingID:       "booking-1",
				SelectedCottage: "Alps",
				SelectedPeriod: &bookingdto.Period{
					Start: time.Date(2026, time.April, 12, 0, 0, 0, 0, time.UTC),
					End:   time.Date(2026, time.April, 15, 0, 0, 0, 0, time.UTC),
				},
			},
		},
	}
	step := BookCottageStep{
		client: resty.New(),
		state:  &domain.State{GuestId: "guest-1"},
		cache:  cache,
	}

	err := step.Execute(context.Background())
	if err == nil || !strings.Contains(err.Error(), "booking already created") {
		t.Fatalf("unexpected error: %v", err)
	}
}
