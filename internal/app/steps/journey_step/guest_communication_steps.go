package journey_step

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/Kenji-Uema/guestSimulator/internal/app/steps"
	"github.com/Kenji-Uema/guestSimulator/internal/config"
	"github.com/Kenji-Uema/guestSimulator/internal/domain"
	"github.com/Kenji-Uema/guestSimulator/internal/domain/dto"
	"github.com/Kenji-Uema/guestSimulator/internal/infra/telemetry"
	"github.com/Kenji-Uema/guestSimulator/internal/port"
	"github.com/Kenji-Uema/guestSimulator/internal/transport/grpc/pb/communication"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel/attribute"
	"google.golang.org/protobuf/proto"
)

const (
	guestCommunicationExchange = "ex.communication"
	guestQueuePrefix           = "q.guest."
	guestRoutingKeyPrefix      = "guest."

	paymentRequestMessageType      = "paymentSimulator.payment.v1.PaymentRequest"
	bookingConfirmationMessageType = "cottageManager.invoice.BookingConfirmedNotificationEvent"
	checkinTomorrowMessageType     = "lodging.v1.CheckInTomorrowNotification"
)

type setupGuestCommunicationStep struct {
	state           *domain.State
	connection      port.RabbitConnection
	consumerFactory port.RabbitConsumerFactory
	runtime         *GuestCommunicationRuntime
}

type waitGuestMessageStep struct {
	name         string
	messageType  string
	state        *domain.State
	cache        port.Cache
	runtime      *GuestCommunicationRuntime
	matchMessage func(ctx context.Context, delivery amqp.Delivery, cacheValue dto.GuestJourneyCacheValue) error
}

type closeGuestCommunicationStep struct {
	state   *domain.State
	runtime *GuestCommunicationRuntime
}

func NewSetupGuestCommunicationStep(state *domain.State, connection port.RabbitConnection, consumerFactory port.RabbitConsumerFactory, runtime *GuestCommunicationRuntime) steps.Step {
	return &setupGuestCommunicationStep{state: state, connection: connection, consumerFactory: consumerFactory, runtime: runtime}
}

func NewWaitPaymentRequestStep(state *domain.State, cache port.Cache, runtime *GuestCommunicationRuntime) steps.Step {
	return &waitGuestMessageStep{
		name:        "WaitPaymentRequestStep",
		messageType: paymentRequestMessageType,
		state:       state,
		cache:       cache,
		runtime:     runtime,
		matchMessage: func(_ context.Context, delivery amqp.Delivery, cacheValue dto.GuestJourneyCacheValue) error {
			var msg communication.PaymentRequest
			if err := proto.Unmarshal(delivery.Body, &msg); err != nil {
				return fmt.Errorf("unmarshal payment request: %w", err)
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
			return nil
		},
	}
}

func NewWaitBookingConfirmationStep(state *domain.State, cache port.Cache, runtime *GuestCommunicationRuntime) steps.Step {
	return &waitGuestMessageStep{
		name:        "WaitBookingConfirmationStep",
		messageType: bookingConfirmationMessageType,
		state:       state,
		cache:       cache,
		runtime:     runtime,
		matchMessage: func(_ context.Context, delivery amqp.Delivery, cacheValue dto.GuestJourneyCacheValue) error {
			var msg communication.BookingConfirmedNotificationEvent
			if err := proto.Unmarshal(delivery.Body, &msg); err != nil {
				return fmt.Errorf("unmarshal booking confirmation: %w", err)
			}
			if cacheValue.Booking == nil {
				return fmt.Errorf("cached booking context is empty")
			}
			if msg.GetBookingId() != cacheValue.Booking.BookingID {
				return fmt.Errorf("booking confirmation bookingId %q does not match cache %q", msg.GetBookingId(), cacheValue.Booking.BookingID)
			}
			if msg.GetBookingStatus() != communication.BookingStatus_BOOKING_STATUS_CONFIRMED {
				return fmt.Errorf("booking confirmation status %q is not confirmed", msg.GetBookingStatus().String())
			}
			if msg.GetGuest() == nil || msg.GetGuest().GetGuestId() != state.GuestId {
				return fmt.Errorf("booking confirmation guestId does not match guest")
			}
			return nil
		},
	}
}

func NewWaitCheckinTomorrowStep(state *domain.State, cache port.Cache, runtime *GuestCommunicationRuntime) steps.Step {
	return &waitGuestMessageStep{
		name:        "WaitCheckinTomorrowStep",
		messageType: checkinTomorrowMessageType,
		state:       state,
		cache:       cache,
		runtime:     runtime,
		matchMessage: func(_ context.Context, delivery amqp.Delivery, cacheValue dto.GuestJourneyCacheValue) error {
			var msg communication.CheckInTomorrowNotification
			if err := proto.Unmarshal(delivery.Body, &msg); err != nil {
				return fmt.Errorf("unmarshal checkin tomorrow: %w", err)
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
		},
	}
}

func NewCloseGuestCommunicationStep(state *domain.State, runtime *GuestCommunicationRuntime) steps.Step {
	return &closeGuestCommunicationStep{state: state, runtime: runtime}
}

func (s setupGuestCommunicationStep) Name() string {
	return "SetupGuestCommunicationStep"
}

func (s setupGuestCommunicationStep) Validate() error {
	if s.state == nil {
		return fmt.Errorf("invalid state, state is nil")
	}
	if s.state.GuestId == "" {
		return fmt.Errorf("invalid state, guestId is empty")
	}
	if s.connection == nil {
		return fmt.Errorf("invalid rabbitmq connection")
	}
	if s.consumerFactory == nil {
		return fmt.Errorf("invalid rabbitmq consumer factory")
	}
	if s.runtime == nil {
		return fmt.Errorf("invalid guest communication runtime")
	}
	return nil
}

func (s setupGuestCommunicationStep) Execute(ctx context.Context) error {
	ctx, span := telemetry.Tracer.Start(ctx, "SetupGuestCommunicationStep")
	defer span.End()

	s.state.QueueName = guestQueuePrefix + s.state.GuestId
	s.state.RoutingKey = guestRoutingKeyPrefix + s.state.GuestId

	consumer, err := s.consumerFactory.NewConsumer(s.connection, config.ConsumeConfig{})
	if err != nil {
		return err
	}

	if err := consumer.DeclareQueue(ctx, config.QueueConfig{
		Name:       s.state.QueueName,
		Durable:    false,
		AutoDelete: true,
		Exclusive:  false,
		NoWait:     false,
	}); err != nil {
		_ = consumer.CloseChannel()
		return err
	}

	if err := consumer.BindQueue(ctx, config.BindingConfig{
		ExchangeName: guestCommunicationExchange,
		RoutingKey:   s.state.RoutingKey,
		NoWait:       false,
	}); err != nil {
		_ = consumer.CloseChannel()
		return err
	}

	deliveries, err := consumer.Consume(ctx)
	if err != nil {
		_ = consumer.CloseChannel()
		return err
	}

	s.runtime.Consumer = consumer
	s.runtime.Deliveries = deliveries
	s.runtime.Pending = nil

	slog.InfoContext(ctx, "guest communication queue ready", "queue", s.state.QueueName, "routingKey", s.state.RoutingKey)
	return nil
}

func (s waitGuestMessageStep) Name() string {
	return s.name
}

func (s waitGuestMessageStep) Validate() error {
	if s.runtime == nil {
		return fmt.Errorf("invalid guest communication runtime")
	}
	if s.state == nil {
		return fmt.Errorf("invalid state, state is nil")
	}
	if s.cache == nil {
		return fmt.Errorf("invalid guest journey cache")
	}
	if s.runtime.Deliveries == nil {
		return fmt.Errorf("guest communication deliveries not initialized")
	}
	if s.matchMessage == nil {
		return fmt.Errorf("guest communication matcher not configured")
	}
	return nil
}

func (s waitGuestMessageStep) Execute(ctx context.Context) error {
	ctx, span := telemetry.Tracer.Start(ctx, s.name)
	defer span.End()

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if delivery, ok := s.takeMatchingPending(ctx); ok {
		return s.ackMatchedDelivery(ctx, delivery)
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case delivery, ok := <-s.runtime.Deliveries:
			if !ok {
				return fmt.Errorf("guest communication channel closed")
			}

			if err := s.matchDelivery(ctx, delivery); err != nil {
				s.runtime.addPending(delivery)
				slog.DebugContext(ctx, "guest communication message buffered for another step",
					"step", s.name,
					"expectedType", s.messageType,
					"routingKey", delivery.RoutingKey,
					"error", err)
				continue
			}

			return s.ackMatchedDelivery(ctx, delivery)
		}
	}
}

func (s closeGuestCommunicationStep) Name() string {
	return "CloseGuestCommunicationStep"
}

func (s closeGuestCommunicationStep) Validate() error {
	if s.state == nil {
		return fmt.Errorf("invalid state, state is nil")
	}
	if s.runtime == nil {
		return fmt.Errorf("invalid guest communication runtime")
	}
	return nil
}

func (s closeGuestCommunicationStep) Execute(ctx context.Context) error {
	ctx, span := telemetry.Tracer.Start(ctx, "DeleteGuestCommunicationQueueStep")
	defer span.End()
	span.SetAttributes(
		attribute.String("guest.queue.name", s.state.QueueName),
		attribute.String("guest.queue.routing_key", s.state.RoutingKey),
		attribute.Bool("guest.queue.auto_delete", true),
	)

	if s.runtime.Consumer == nil {
		return nil
	}

	if err := s.runtime.Consumer.CloseChannel(); err != nil {
		return err
	}

	s.runtime.Consumer = nil
	s.runtime.Deliveries = nil
	s.runtime.Pending = nil
	slog.InfoContext(ctx, "guest communication queue closed", "queue", s.state.QueueName, "routingKey", s.state.RoutingKey)
	return nil
}

func (s waitGuestMessageStep) takeMatchingPending(ctx context.Context) (amqp.Delivery, bool) {
	s.runtime.mu.Lock()
	defer s.runtime.mu.Unlock()

	for idx, delivery := range s.runtime.Pending {
		if err := s.matchDelivery(ctx, delivery); err != nil {
			continue
		}

		s.runtime.Pending = append(s.runtime.Pending[:idx], s.runtime.Pending[idx+1:]...)
		return delivery, true
	}

	return amqp.Delivery{}, false
}

func (r *GuestCommunicationRuntime) addPending(delivery amqp.Delivery) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.Pending = append(r.Pending, delivery)
}

func (s waitGuestMessageStep) matchDelivery(ctx context.Context, delivery amqp.Delivery) error {
	if messageType := deliveryHeader(delivery, "message_type"); messageType != s.messageType {
		return fmt.Errorf("message_type %q does not match expected %q", messageType, s.messageType)
	}

	cacheValue, err := s.cache.Load(ctx, s.state)
	if err != nil {
		return err
	}

	return s.matchMessage(ctx, delivery, cacheValue)
}

func (s waitGuestMessageStep) ackMatchedDelivery(ctx context.Context, delivery amqp.Delivery) error {
	slog.InfoContext(ctx, "guest communication message received",
		"step", s.name,
		"queue", s.state.QueueName,
		"routingKey", delivery.RoutingKey,
		"messageType", deliveryHeader(delivery, "message_type"),
		"body", string(delivery.Body))

	if err := delivery.Ack(false); err != nil {
		return err
	}

	return nil
}

func deliveryHeader(delivery amqp.Delivery, key string) string {
	if delivery.Headers == nil {
		return ""
	}

	value, ok := delivery.Headers[key]
	if !ok || value == nil {
		return ""
	}

	switch typed := value.(type) {
	case string:
		return typed
	case []byte:
		return string(typed)
	default:
		return fmt.Sprint(typed)
	}
}
