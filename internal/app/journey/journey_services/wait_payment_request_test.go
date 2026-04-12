package journey_services

import (
	"context"
	"testing"

	"github.com/Kenji-Uema/guestSimulator/internal/domain"
	"github.com/Kenji-Uema/guestSimulator/internal/domain/dto"
	"github.com/Kenji-Uema/guestSimulator/internal/domain/dto/communication"
	"github.com/Kenji-Uema/guestSimulator/internal/domain/dto/guest_registration"
	redisfakes "github.com/Kenji-Uema/guestSimulator/internal/infra/redis/fakes"
)

func TestSubscribeAndTakePendingPaymentRequest(t *testing.T) {
	bus := NewGuestCommunicationBus()
	ch := make(chan *communication.PaymentRequest, 1)
	unsubscribe := subscribePaymentRequest(bus, ch)

	if len(bus.paymentRequestChannels.Values()) != 1 {
		t.Fatalf("unexpected subscriber count: %d", len(bus.paymentRequestChannels.Values()))
	}

	unsubscribe()

	if len(bus.paymentRequestChannels.Values()) != 0 {
		t.Fatalf("unexpected subscriber count after unsubscribe: %d", len(bus.paymentRequestChannels.Values()))
	}

	expected := &communication.PaymentRequest{InvoiceNumber: "invoice-1"}
	bus.pendingPaymentRequests = []*communication.PaymentRequest{expected}

	got, ok := takePendingPaymentRequest(bus)
	if !ok || got != expected {
		t.Fatalf("unexpected pending payment request: %#v %v", got, ok)
	}
}

func TestMatchPaymentRequestSuccess(t *testing.T) {
	cache := &redisfakes.Cache{
		LoadValue: dto.GuestJourneyCacheValue{
			PersonalInfo: &guest_registration.Guest{Email: "guest@test.com"},
			Booking:      &dto.GuestJourneyBooking{SelectedCottage: "Alps"},
		},
	}

	err := matchPaymentRequest(context.Background(), &domain.State{}, cache, &communication.PaymentRequest{
		InvoiceNumber: "invoice-1",
		Total:         &communication.Money{Amount: 100, Currency: "USD"},
		Booking:       &communication.BookingSummary{CottageName: "Alps"},
		Payer:         &communication.PayerSummary{Email: "guest@test.com"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cache.SavedValue.Invoice == nil || cache.SavedValue.Invoice.InvoiceNumber != "invoice-1" {
		t.Fatalf("unexpected saved invoice: %#v", cache.SavedValue.Invoice)
	}
}
