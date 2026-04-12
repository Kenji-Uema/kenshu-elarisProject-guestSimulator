package journey_services

import (
	"context"
	"strings"
	"testing"

	"github.com/Kenji-Uema/guestSimulator/internal/config"
	"github.com/Kenji-Uema/guestSimulator/internal/domain"
	mqfakes "github.com/Kenji-Uema/guestSimulator/internal/infra/mq/fakes"
	amqp "github.com/rabbitmq/amqp091-go"
)

func TestSetupCommunicationRejectsInvalidDependencies(t *testing.T) {
	bus := NewGuestCommunicationBus()

	err := SetupCommunication(context.Background(), &domain.State{}, &mqfakes.Consumer{}, config.RabbitMqConsumerConfig{}, bus)
	if err == nil || !strings.Contains(err.Error(), "guestId is empty") {
		t.Fatalf("unexpected guest-id error: %v", err)
	}

	err = SetupCommunication(context.Background(), &domain.State{GuestId: "guest-1"}, nil, config.RabbitMqConsumerConfig{}, bus)
	if err == nil || !strings.Contains(err.Error(), "invalid rabbitmq consumer") {
		t.Fatalf("unexpected consumer error: %v", err)
	}
}

func TestAckDeliveryAndNackDeliveryCallAcknowledger(t *testing.T) {
	acker := &mqfakes.Acknowledger{}
	delivery := amqp.Delivery{
		Acknowledger: acker,
		DeliveryTag:  1,
	}

	ackDelivery(context.Background(), delivery, "delivery")
	nackDelivery(context.Background(), delivery, "delivery")

	if acker.AckCalls != 1 {
		t.Fatalf("unexpected ack calls: %d", acker.AckCalls)
	}
	if acker.NackCalls != 1 {
		t.Fatalf("unexpected nack calls: %d", acker.NackCalls)
	}
}
