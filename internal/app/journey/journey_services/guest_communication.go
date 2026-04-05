package journey_services

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Kenji-Uema/guestSimulator/internal/config"
	"github.com/Kenji-Uema/guestSimulator/internal/domain"
	"github.com/Kenji-Uema/guestSimulator/internal/port"
	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	guestCommunicationExchange = "ex.communication"
	guestQueuePrefix           = "q.guest."
	guestRoutingKeyPrefix      = "guest."

	paymentRequestMessageType      = "paymentSimulator.payment.v1.PaymentRequest"
	bookingConfirmationMessageType = "cottageManager.invoice.BookingConfirmedNotificationEvent"
	checkinTomorrowMessageType     = "lodging.v1.CheckInTomorrowNotification"
)

func SetupCommunication(ctx context.Context, state *domain.State, consumer port.RabbitConsumer, bus *GuestCommunicationBus) error {

	if state == nil {
		return fmt.Errorf("invalid state, state is nil")
	}
	if state.GuestId == "" {
		return fmt.Errorf("invalid state, guestId is empty")
	}
	if consumer == nil {
		return fmt.Errorf("invalid rabbitmq consumer")
	}
	if bus == nil {
		return fmt.Errorf("invalid guest communication runtime")
	}

	state.QueueName = guestQueuePrefix + state.GuestId
	state.RoutingKey = guestRoutingKeyPrefix + state.GuestId

	if err := consumer.DeclareQueue(ctx, config.QueueConfig{
		Name:       state.QueueName,
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
		RoutingKey:   state.RoutingKey,
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

	bus.Reset()
	bus.Consumer = consumer
	bus.Start(ctx, deliveries)

	slog.InfoContext(ctx, "guest communication queue ready", "queue", state.QueueName, "routingKey", state.RoutingKey)
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

func ackDelivery(ctx context.Context, delivery amqp.Delivery, deliveryName string) {
	if err := delivery.Ack(false); err != nil {
		slog.ErrorContext(ctx, "failed to ack "+deliveryName, "error", err, "routingKey", delivery.RoutingKey)
	}
}

func nackDelivery(ctx context.Context, delivery amqp.Delivery, deliveryName string) {
	if err := delivery.Nack(false, false); err != nil {
		slog.ErrorContext(ctx, "failed to nack "+deliveryName, "error", err, "routingKey", delivery.RoutingKey)
	}
}
