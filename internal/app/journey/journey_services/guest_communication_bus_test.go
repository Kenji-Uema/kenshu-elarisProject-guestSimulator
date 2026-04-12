package journey_services

import (
	"context"
	"testing"

	"github.com/Kenji-Uema/guestSimulator/internal/domain/dto/communication"
)

func TestNewGuestCommunicationBusInitializesChannels(t *testing.T) {
	bus := NewGuestCommunicationBus()

	if bus == nil {
		t.Fatal("expected bus")
	}
	if bus.paymentRequestChannels == nil || bus.bookingConfirmationChannels == nil || bus.checkinTodayChannels == nil {
		t.Fatalf("unexpected bus channels: %#v", bus)
	}
	if bus.Done() == nil {
		t.Fatal("expected done channel")
	}
}

func TestGuestCommunicationBusPublishPaymentRequestQueuesWithoutSubscriber(t *testing.T) {
	bus := NewGuestCommunicationBus()
	msg := &communication.PaymentRequest{InvoiceNumber: "invoice-1"}

	bus.publishPaymentRequest(context.Background(), msg)

	if len(bus.pendingPaymentRequests) != 1 {
		t.Fatalf("unexpected pending requests: %#v", bus.pendingPaymentRequests)
	}
}

func TestGuestCommunicationBusPublishPaymentRequestDeliversToSubscriber(t *testing.T) {
	bus := NewGuestCommunicationBus()
	ch := make(chan *communication.PaymentRequest, 1)
	unsubscribe := subscribePaymentRequest(bus, ch)
	defer unsubscribe()

	msg := &communication.PaymentRequest{InvoiceNumber: "invoice-1"}
	bus.publishPaymentRequest(context.Background(), msg)

	select {
	case got := <-ch:
		if got.GetInvoiceNumber() != "invoice-1" {
			t.Fatalf("unexpected message: %#v", got)
		}
	default:
		t.Fatal("expected payment request delivery")
	}
}

func TestGuestCommunicationBusPublishCheckinTodayQueuesWithoutSubscriber(t *testing.T) {
	bus := NewGuestCommunicationBus()
	msg := &communication.CheckInTodayNotification{BookingId: "booking-1"}

	bus.publishCheckinToday(context.Background(), msg)

	if len(bus.pendingCheckinToday) != 1 {
		t.Fatalf("unexpected pending notifications: %#v", bus.pendingCheckinToday)
	}
}

func TestGuestCommunicationBusPublishCheckinTodayDeliversToSubscriber(t *testing.T) {
	bus := NewGuestCommunicationBus()
	ch := make(chan *communication.CheckInTodayNotification, 1)
	unsubscribe := subscribeCheckinToday(bus, ch)
	defer unsubscribe()

	msg := &communication.CheckInTodayNotification{BookingId: "booking-1"}
	bus.publishCheckinToday(context.Background(), msg)

	select {
	case got := <-ch:
		if got.GetBookingId() != "booking-1" {
			t.Fatalf("unexpected message: %#v", got)
		}
	default:
		t.Fatal("expected checkin notification delivery")
	}
}

func TestGuestCommunicationBusErrReturnsLifecycleError(t *testing.T) {
	bus := NewGuestCommunicationBus()
	bus.lifecycleMu.Lock()
	bus.err = context.Canceled
	bus.lifecycleMu.Unlock()

	if err := bus.Err(); err != context.Canceled {
		t.Fatalf("unexpected err: %v", err)
	}
}
