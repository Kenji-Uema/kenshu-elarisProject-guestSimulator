package port

import (
	"context"

	"github.com/Kenji-Uema/guestSimulator/internal/config"
	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitConnection interface {
	IsConnectionOpen() bool
	Close() error
}

type RabbitConsumer interface {
	DeclareQueue(ctx context.Context, cfg config.QueueConfig) error
	BindQueue(ctx context.Context, cfg config.BindingConfig) error
	Consume(ctx context.Context) (<-chan amqp.Delivery, error)
	CloseChannel() error
}
