package mq

import (
	"context"
	"fmt"

	"github.com/Kenji-Uema/guestSimulator/internal/config"
	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitmqConsumer struct {
	*RabbitMqChannel
	queue         string
	consumeConfig config.ConsumeConfig
}

func NewRabbitmqConsumer(rabbitMqConnection *RabbitMqConnection, consumeConfig config.ConsumeConfig) (*RabbitmqConsumer, error) {
	consumer := &RabbitmqConsumer{
		RabbitMqChannel: NewRabbitMqChannel(rabbitMqConnection),
		consumeConfig:   consumeConfig,
	}
	if err := consumer.openChannel(); err != nil {
		return nil, err
	}

	return consumer, nil
}

func (c *RabbitmqConsumer) DeclareQueue(ctx context.Context, cfg config.QueueConfig) error {
	if c.channel == nil || c.channel.IsClosed() {
		if err := c.reopenChannel(ctx); err != nil {
			return err
		}
	}

	q, err := c.channel.QueueDeclare(
		cfg.Name,
		cfg.Durable,
		cfg.AutoDelete,
		cfg.Exclusive,
		cfg.NoWait,
		nil,
	)
	if err != nil {
		return fmt.Errorf("declare queue %q: %w", cfg.Name, err)
	}

	c.queue = q.Name
	return nil
}

func (c *RabbitmqConsumer) BindQueue(ctx context.Context, cfg config.BindingConfig) error {
	if c.channel == nil || c.channel.IsClosed() {
		if err := c.reopenChannel(ctx); err != nil {
			return err
		}
	}

	if c.queue == "" {
		return fmt.Errorf("queue not declared")
	}

	if err := c.channel.QueueBind(c.queue, cfg.RoutingKey, cfg.ExchangeName, cfg.NoWait, nil); err != nil {
		return fmt.Errorf("bind queue %q to exchange %q with routing key %q: %w", c.queue, cfg.ExchangeName, cfg.RoutingKey, err)
	}

	return nil
}

func (c *RabbitmqConsumer) Consume(ctx context.Context) (<-chan amqp.Delivery, error) {
	if c.channel == nil || c.channel.IsClosed() {
		if err := c.reopenChannel(ctx); err != nil {
			return nil, err
		}
	}

	return c.channel.ConsumeWithContext(
		ctx,
		c.queue,
		c.consumeConfig.Consumer,
		c.consumeConfig.AutoAck,
		c.consumeConfig.Exclusive,
		c.consumeConfig.NoLocal,
		c.consumeConfig.NoWait,
		nil,
	)
}
