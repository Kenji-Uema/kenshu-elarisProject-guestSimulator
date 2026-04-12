package fakes

import (
	"context"

	"github.com/Kenji-Uema/guestSimulator/internal/config"
	amqp "github.com/rabbitmq/amqp091-go"
)

type Consumer struct {
	DeclareErr error
	BindErr    error
	ConsumeErr error
	CloseErr   error

	DeclareCfg config.QueueConfig
	BindingCfg config.BindingConfig
	Deliveries chan amqp.Delivery
	CloseCalls int
}

func (c *Consumer) DeclareQueue(_ context.Context, cfg config.QueueConfig) error {
	c.DeclareCfg = cfg
	return c.DeclareErr
}

func (c *Consumer) BindQueue(_ context.Context, cfg config.BindingConfig) error {
	c.BindingCfg = cfg
	return c.BindErr
}

func (c *Consumer) Consume(context.Context) (<-chan amqp.Delivery, error) {
	return c.Deliveries, c.ConsumeErr
}

func (c *Consumer) CloseChannel() error {
	c.CloseCalls++
	return c.CloseErr
}
