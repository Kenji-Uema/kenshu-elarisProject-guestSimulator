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

	checkinTomorrowMu       sync.RWMutex
	checkinTomorrowChannels *domain.Set[chan<- *communication.CheckInTomorrowNotification]
	pendingCheckinTomorrow  []*communication.CheckInTomorrowNotification

	lifecycleMu sync.RWMutex
	done        chan struct{}
	err         error
}

func NewGuestCommunicationBus() *GuestCommunicationBus {
	return &GuestCommunicationBus{
		paymentRequestChannels:      domain.NewSet[chan<- *communication.PaymentRequest](),
		bookingConfirmationChannels: domain.NewSet[chan<- *communication.BookingConfirmedNotificationEvent](),
		checkinTomorrowChannels:     domain.NewSet[chan<- *communication.CheckInTomorrowNotification](),
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

	b.checkinTomorrowMu.Lock()
	b.pendingCheckinTomorrow = nil
	b.checkinTomorrowChannels = domain.NewSet[chan<- *communication.CheckInTomorrowNotification]()
	b.checkinTomorrowMu.Unlock()

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
		b.publishPaymentRequest(ctx, &msg)
		return nil
	case bookingConfirmationMessageType:
		var msg communication.BookingConfirmedNotificationEvent
		if err := proto.Unmarshal(delivery.Body, &msg); err != nil {
			return fmt.Errorf("unmarshal booking confirmation: %w", err)
		}
		b.publishBookingConfirmation(ctx, &msg)
		return nil
	case checkinTomorrowMessageType:
		var msg communication.CheckInTomorrowNotification
		if err := proto.Unmarshal(delivery.Body, &msg); err != nil {
			return fmt.Errorf("unmarshal checkin tomorrow: %w", err)
		}
		b.publishCheckinTomorrow(ctx, &msg)
		return nil
	default:
		return fmt.Errorf("unsupported message_type %q", deliveryHeader(delivery, "message_type"))
	}
}

func (b *GuestCommunicationBus) publishPaymentRequest(ctx context.Context, msg *communication.PaymentRequest) {
	b.paymentRequestMu.Lock()
	defer b.paymentRequestMu.Unlock()

	if !notifyPaymentRequestSubscribers(ctx, b.paymentRequestChannels.Values(), msg) {
		b.pendingPaymentRequests = append(b.pendingPaymentRequests, msg)
	}
}

func (b *GuestCommunicationBus) publishBookingConfirmation(ctx context.Context, msg *communication.BookingConfirmedNotificationEvent) {
	b.bookingConfirmationMu.Lock()
	defer b.bookingConfirmationMu.Unlock()

	if !notifyBookingConfirmationSubscribers(ctx, b.bookingConfirmationChannels.Values(), msg) {
		b.pendingBookingConfirmations = append(b.pendingBookingConfirmations, msg)
	}
}

func (b *GuestCommunicationBus) publishCheckinTomorrow(ctx context.Context, msg *communication.CheckInTomorrowNotification) {
	b.checkinTomorrowMu.Lock()
	defer b.checkinTomorrowMu.Unlock()

	if !notifyCheckinTomorrowSubscribers(ctx, b.checkinTomorrowChannels.Values(), msg) {
		b.pendingCheckinTomorrow = append(b.pendingCheckinTomorrow, msg)
	}
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

func notifyCheckinTomorrowSubscribers(ctx context.Context, channels []chan<- *communication.CheckInTomorrowNotification, msg *communication.CheckInTomorrowNotification) bool {
	delivered := false
	for _, ch := range channels {
		select {
		case ch <- msg:
			delivered = true
			slog.DebugContext(ctx, "published guest communication event to subscriber", "event", "checkin tomorrow")
		default:
			slog.WarnContext(ctx, "guest communication subscriber is busy", "event", "checkin tomorrow")
		}
	}

	return delivered
}
