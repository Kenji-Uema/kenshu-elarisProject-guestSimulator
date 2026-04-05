package journey_services

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Kenji-Uema/guestSimulator/internal/domain"
	"github.com/Kenji-Uema/guestSimulator/internal/domain/dto"
	"github.com/Kenji-Uema/guestSimulator/internal/domain/dto/communication"
	"github.com/Kenji-Uema/guestSimulator/internal/port"
)

func WaitPaymentRequestMessage(ctx context.Context, state *domain.State, cache port.Cache, bus *GuestCommunicationBus) error {
	ch := make(chan *communication.PaymentRequest, 1)
	unsubscribeFn := subscribePaymentRequest(bus, ch)
	defer unsubscribeFn()

	for {
		if msg, ok := takePendingPaymentRequest(bus); ok {
			if err := matchPaymentRequest(ctx, state, cache, msg); err == nil {
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
			if err := matchPaymentRequest(ctx, state, cache, msg); err == nil {
				return nil
			}
		}
	}
}

func matchPaymentRequest(ctx context.Context, state *domain.State, cache port.Cache, msg *communication.PaymentRequest) error {
	if msg == nil {
		return fmt.Errorf("payment request is nil")
	}
	cacheValue, err := cache.Load(ctx, state)
	if err != nil {
		return err
	}
	if strings.TrimSpace(msg.GetInvoiceNumber()) == "" {
		return fmt.Errorf("payment request missing invoiceNumber")
	}
	if msg.GetTotal() == nil || msg.GetTotal().GetAmount() <= 0 || strings.TrimSpace(msg.GetTotal().GetCurrency()) == "" {
		return fmt.Errorf("payment request has invalid total")
	}
	if msg.GetBooking() == nil || strings.TrimSpace(msg.GetBooking().GetCottageName()) == "" {
		return fmt.Errorf("payment request missing booking summary")
	}
	if cacheValue.PersonalInfo == nil {
		return fmt.Errorf("cached guest context is empty")
	}
	if cacheValue.Booking == nil || cacheValue.Booking.SelectedCottage == "" {
		return fmt.Errorf("cached booking context is empty")
	}
	if msg.GetPayer() == nil || !strings.EqualFold(strings.TrimSpace(msg.GetPayer().GetEmail()), strings.TrimSpace(cacheValue.PersonalInfo.Email)) {
		return fmt.Errorf("payment request payer does not match guest email")
	}
	if msg.GetBooking().GetCottageName() != cacheValue.Booking.SelectedCottage {
		return fmt.Errorf("payment request cottage %q does not match selected cottage %q", msg.GetBooking().GetCottageName(), cacheValue.Booking.SelectedCottage)
	}

	if cacheValue.Invoice == nil {
		cacheValue.Invoice = &dto.GuestJourneyInvoice{}
	}
	cacheValue.Invoice.InvoiceNumber = strings.TrimSpace(msg.GetInvoiceNumber())
	if err := cache.Save(ctx, state, cacheValue); err != nil {
		return err
	}

	return nil
}

func subscribePaymentRequest(bus *GuestCommunicationBus, ch chan<- *communication.PaymentRequest) func() {
	bus.paymentRequestMu.Lock()
	bus.paymentRequestChannels.Add(ch)
	bus.paymentRequestMu.Unlock()

	return func() {
		bus.paymentRequestMu.Lock()
		bus.paymentRequestChannels.Remove(ch)
		bus.paymentRequestMu.Unlock()
	}
}

func takePendingPaymentRequest(bus *GuestCommunicationBus) (*communication.PaymentRequest, bool) {
	bus.paymentRequestMu.Lock()
	defer bus.paymentRequestMu.Unlock()
	if len(bus.pendingPaymentRequests) == 0 {
		return nil, false
	}

	msg := bus.pendingPaymentRequests[0]
	bus.pendingPaymentRequests = bus.pendingPaymentRequests[1:]
	return msg, true
}
