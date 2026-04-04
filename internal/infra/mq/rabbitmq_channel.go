package mq

import (
	"context"
	"log/slog"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMqChannel struct {
	*RabbitMqConnection
	channel *amqp.Channel
}

func NewRabbitMqChannel(rabbitmqConnection *RabbitMqConnection) *RabbitMqChannel {
	return &RabbitMqChannel{RabbitMqConnection: rabbitmqConnection}
}

func (r *RabbitMqChannel) CloseChannel() error {
	if r.channel == nil || r.channel.IsClosed() {
		return nil
	}

	return r.channel.Close()
}

func (r *RabbitMqChannel) openChannel() error {
	conn, err := r.openConnection()
	if err != nil {
		return err
	}

	ch, err := conn.Channel()
	if err != nil {
		if !conn.IsClosed() {
			return err
		}

		conn, err = r.openConnection()
		if err != nil {
			return err
		}

		ch, err = conn.Channel()
		if err != nil {
			return err
		}
	}

	r.channel = ch
	return nil
}

func (r *RabbitMqChannel) reopenChannel(ctx context.Context) error {
	slog.WarnContext(ctx, "channel is closed, opening a new one")
	return r.openChannel()
}
