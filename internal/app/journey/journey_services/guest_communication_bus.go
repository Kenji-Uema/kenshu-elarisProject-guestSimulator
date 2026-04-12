package journey_services

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/Kenji-Uema/guestSimulator/internal/domain"
	"github.com/Kenji-Uema/guestSimulator/internal/domain/dto/communication"
	"github.com/Kenji-Uema/guestSimulator/internal/port"
	amqp "github.com/rabbitmq/amqp091-go"
	"google.golang.org/protobuf/proto"
)

type GuestCommunicationBus struct {
	Consumer port.RabbitConsumer

	paymentRequestMu       sync.RWMutex
	paymentRequestChannels *domain.Set[chan<- *communication.PaymentRequest]
	pendingPaymentRequests []*communication.PaymentRequest

	bookingConfirmationMu       sync.RWMutex
	bookingConfirmationChannels *domain.Set[chan<- *communication.BookingConfirmedNotificationEvent]
	pendingBookingConfirmations []*communication.BookingConfirmedNotificationEvent

	checkinTodayMu       sync.RWMutex
	checkinTodayChannels *domain.Set[chan<- *communication.CheckInTodayNotification]
	pendingCheckinToday  []*communication.CheckInTodayNotification

	lifecycleMu sync.RWMutex
	done        chan struct{}
	err         error
}

func NewGuestCommunicationBus() *GuestCommunicationBus {
	return &GuestCommunicationBus{
		paymentRequestChannels:      domain.NewSet[chan<- *communication.PaymentRequest](),
		bookingConfirmationChannels: domain.NewSet[chan<- *communication.BookingConfirmedNotificationEvent](),
		checkinTodayChannels:        domain.NewSet[chan<- *communication.CheckInTodayNotification](),
		done:                        make(chan struct{}),
	}
}

func (b *GuestCommunicationBus) Start(ctx context.Context, deliveries <-chan amqp.Delivery) {
	b.lifecycleMu.Lock()
	b.err = nil
	b.done = make(chan struct{})
	done := b.done
	b.lifecycleMu.Unlock()

	go func() {
		defer close(done)

		for {
			select {
			case <-ctx.Done():
				b.lifecycleMu.Lock()
				b.err = ctx.Err()
				b.lifecycleMu.Unlock()
				return
			case delivery, ok := <-deliveries:
				if !ok {
					b.lifecycleMu.Lock()
					b.err = fmt.Errorf("guest communication channel closed")
					b.lifecycleMu.Unlock()
					return
				}

				if err := b.handleDelivery(ctx, delivery); err != nil {
					slog.ErrorContext(ctx, "failed to process guest communication event", "error", err, "routingKey", delivery.RoutingKey)
					nackDelivery(ctx, delivery, "guestCommunicationEvent")
					continue
				}

				ackDelivery(ctx, delivery, "guestCommunicationEvent")
			}
		}
	}()
}

func (b *GuestCommunicationBus) Done() <-chan struct{} {
	b.lifecycleMu.RLock()
	defer b.lifecycleMu.RUnlock()
	return b.done
}

func (b *GuestCommunicationBus) Err() error {
	b.lifecycleMu.RLock()
	defer b.lifecycleMu.RUnlock()
	return b.err
}

func (b *GuestCommunicationBus) Reset() {
	b.paymentRequestMu.Lock()
	b.pendingPaymentRequests = nil
	b.paymentRequestChannels = domain.NewSet[chan<- *communication.PaymentRequest]()
	b.paymentRequestMu.Unlock()

	b.bookingConfirmationMu.Lock()
	b.pendingBookingConfirmations = nil
	b.bookingConfirmationChannels = domain.NewSet[chan<- *communication.BookingConfirmedNotificationEvent]()
	b.bookingConfirmationMu.Unlock()

	b.checkinTodayMu.Lock()
	b.pendingCheckinToday = nil
	b.checkinTodayChannels = domain.NewSet[chan<- *communication.CheckInTodayNotification]()
	b.checkinTodayMu.Unlock()

	b.lifecycleMu.Lock()
	b.err = nil
	b.done = make(chan struct{})
	b.lifecycleMu.Unlock()
}

func (b *GuestCommunicationBus) handleDelivery(ctx context.Context, delivery amqp.Delivery) error {
	switch deliveryHeader(delivery, "message_type") {
	case paymentRequestMessageType:
		var msg communication.PaymentRequest
		if err := proto.Unmarshal(delivery.Body, &msg); err != nil {
			return fmt.Errorf("unmarshal payment request: %w", err)
		}
		slog.InfoContext(ctx, "payment request message received",
			"invoiceNumber", msg.GetInvoiceNumber(),
			"payerEmail", msg.GetPayer().GetEmail(),
			"cottageName", msg.GetBooking().GetCottageName(),
			"routingKey", delivery.RoutingKey)
		b.publishPaymentRequest(ctx, &msg)
		return nil
	case bookingConfirmationMessageType:
		var msg communication.BookingConfirmedNotificationEvent
		if err := proto.Unmarshal(delivery.Body, &msg); err != nil {
			return fmt.Errorf("unmarshal booking confirmation: %w", err)
		}
		slog.InfoContext(ctx, "booking confirmation message received",
			"bookingId", msg.GetBookingId(),
			"guestId", msg.GetGuest().GetGuestId(),
			"status", msg.GetBookingStatus().String(),
			"routingKey", delivery.RoutingKey)
		b.publishBookingConfirmation(ctx, &msg)
		return nil
	case checkinTodayMessageType:
		var msg communication.CheckInTodayNotification
		if err := proto.Unmarshal(delivery.Body, &msg); err != nil {
			return fmt.Errorf("unmarshal checkin today: %w", err)
		}
		slog.InfoContext(ctx, "checkin today message received",
			"bookingId", msg.GetBookingId(),
			"guestId", msg.GetGuestId(),
			"cottageName", msg.GetCottageName(),
			"checkIn", msg.GetCheckIn().AsTime().UTC(),
			"routingKey", delivery.RoutingKey)
		b.publishCheckinToday(ctx, &msg)
		return nil
	default:
		return fmt.Errorf("unsupported message_type %q", deliveryHeader(delivery, "message_type"))
	}
}

func (b *GuestCommunicationBus) publishPaymentRequest(ctx context.Context, msg *communication.PaymentRequest) {
	b.paymentRequestMu.Lock()
	defer b.paymentRequestMu.Unlock()

	subscriberCount := len(b.paymentRequestChannels.Values())
	if !notifyPaymentRequestSubscribers(ctx, b.paymentRequestChannels.Values(), msg) {
		b.pendingPaymentRequests = append(b.pendingPaymentRequests, msg)
		slog.InfoContext(ctx, "payment request queued for later processing",
			"invoiceNumber", msg.GetInvoiceNumber(),
			"pendingCount", len(b.pendingPaymentRequests),
			"subscriberCount", subscriberCount)
		return
	}
	slog.InfoContext(ctx, "payment request delivered to active subscriber",
		"invoiceNumber", msg.GetInvoiceNumber(),
		"subscriberCount", subscriberCount)
}

func (b *GuestCommunicationBus) publishBookingConfirmation(ctx context.Context, msg *communication.BookingConfirmedNotificationEvent) {
	b.bookingConfirmationMu.Lock()
	defer b.bookingConfirmationMu.Unlock()

	subscriberCount := len(b.bookingConfirmationChannels.Values())
	if !notifyBookingConfirmationSubscribers(ctx, b.bookingConfirmationChannels.Values(), msg) {
		b.pendingBookingConfirmations = append(b.pendingBookingConfirmations, msg)
		slog.InfoContext(ctx, "booking confirmation queued for later processing",
			"bookingId", msg.GetBookingId(),
			"guestId", msg.GetGuest().GetGuestId(),
			"pendingCount", len(b.pendingBookingConfirmations),
			"subscriberCount", subscriberCount)
		return
	}
	slog.InfoContext(ctx, "booking confirmation delivered to active subscriber",
		"bookingId", msg.GetBookingId(),
		"guestId", msg.GetGuest().GetGuestId(),
		"subscriberCount", subscriberCount)
}

func (b *GuestCommunicationBus) publishCheckinToday(ctx context.Context, msg *communication.CheckInTodayNotification) {
	b.checkinTodayMu.Lock()
	defer b.checkinTodayMu.Unlock()

	subscriberCount := len(b.checkinTodayChannels.Values())
	if !notifyCheckinTodaySubscribers(ctx, b.checkinTodayChannels.Values(), msg) {
		b.pendingCheckinToday = append(b.pendingCheckinToday, msg)
		slog.InfoContext(ctx, "checkin today queued for later processing",
			"bookingId", msg.GetBookingId(),
			"guestId", msg.GetGuestId(),
			"pendingCount", len(b.pendingCheckinToday),
			"subscriberCount", subscriberCount)
		return
	}
	slog.InfoContext(ctx, "checkin today delivered to active subscriber",
		"bookingId", msg.GetBookingId(),
		"guestId", msg.GetGuestId(),
		"subscriberCount", subscriberCount)
}

func notifyPaymentRequestSubscribers(ctx context.Context, channels []chan<- *communication.PaymentRequest, msg *communication.PaymentRequest) bool {
	delivered := false
	for _, ch := range channels {
		select {
		case ch <- msg:
			delivered = true
			slog.DebugContext(ctx, "published guest communication event to subscriber", "event", "payment request")
		default:
			slog.WarnContext(ctx, "guest communication subscriber is busy", "event", "payment request")
		}
	}

	return delivered
}

func notifyBookingConfirmationSubscribers(ctx context.Context, channels []chan<- *communication.BookingConfirmedNotificationEvent, msg *communication.BookingConfirmedNotificationEvent) bool {
	delivered := false
	for _, ch := range channels {
		select {
		case ch <- msg:
			delivered = true
			slog.DebugContext(ctx, "published guest communication event to subscriber", "event", "booking confirmation")
		default:
			slog.WarnContext(ctx, "guest communication subscriber is busy", "event", "booking confirmation")
		}
	}

	return delivered
}

func notifyCheckinTodaySubscribers(ctx context.Context, channels []chan<- *communication.CheckInTodayNotification, msg *communication.CheckInTodayNotification) bool {
	delivered := false
	for _, ch := range channels {
		select {
		case ch <- msg:
			delivered = true
			slog.DebugContext(ctx, "published guest communication event to subscriber", "event", "checkin today")
		default:
			slog.WarnContext(ctx, "guest communication subscriber is busy", "event", "checkin today")
		}
	}

	return delivered
}
