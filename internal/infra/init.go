package infra

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Kenji-Uema/guestSimulator/internal/config"
	"github.com/Kenji-Uema/guestSimulator/internal/infra/mq"
	redisc "github.com/Kenji-Uema/guestSimulator/internal/infra/redis"
	goredis "github.com/redis/go-redis/v9"
)

type Rabbitmq struct {
	Connection        *mq.RabbitMqConnection
	HourEventConsumer *mq.RabbitmqConsumer
	ConnectionClose   func(context.Context) error
}

type Redis struct {
	Client *redisc.Cache
	Raw    *redisc.Redis
	Close  func() error
}

func NewRabbitmq(ctx context.Context, connCfg config.RabbitMqConnConfig, hourChangeCfg config.RabbitMqConsumerConfig) (Rabbitmq, error) {
	cleanup := make([]func(context.Context) error, 0, 2)

	connection, err := mq.NewRabbitMqConnection(ctx, connCfg)
	if err != nil {
		return Rabbitmq{}, err
	}
	cleanup = append(cleanup, func(context.Context) error {
		return connection.Close()
	})

	consumer, err := mq.NewRabbitmqConsumer(connection, hourChangeCfg.Consume)
	if err != nil {
		_ = runCleanup(ctx, cleanup)
		return Rabbitmq{}, err
	}
	cleanup = append(cleanup, func(context.Context) error {
		return consumer.CloseChannel()
	})

	if err := consumer.DeclareQueue(ctx, hourChangeCfg.Queue); err != nil {
		_ = runCleanup(ctx, cleanup)
		return Rabbitmq{}, fmt.Errorf("declare hour event queue: %w", err)
	}

	if err := consumer.BindQueue(ctx, hourChangeCfg.Binding); err != nil {
		_ = runCleanup(ctx, cleanup)
		return Rabbitmq{}, fmt.Errorf("bind hour event queue: %w", err)
	}

	return Rabbitmq{
		Connection:        connection,
		HourEventConsumer: consumer,
		ConnectionClose:   func(ctx context.Context) error { return runCleanup(ctx, cleanup) },
	}, nil
}

func NewRedisClient(ctx context.Context, cfg config.RedisConfig) (Redis, error) {
	client := goredis.NewClient(&goredis.Options{
		Addr:         fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Username:     string(cfg.Username),
		Password:     string(cfg.Password),
		DB:           cfg.DB,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	})

	redisRaw := redisc.NewRedisClient(client)
	if err := redisRaw.Ping(ctx); err != nil {
		_ = client.Close()
		return Redis{}, fmt.Errorf("redis ping failed for address %s:%d: %w", cfg.Host, cfg.Port, err)
	}
	redisClient := redisc.NewCache(redisRaw)

	return Redis{
		Client: redisClient,
		Raw:    redisRaw,
		Close:  redisRaw.Close,
	}, nil
}

func runCleanup(ctx context.Context, cleanup []func(context.Context) error) error {
	var shutdownErr error
	for i := len(cleanup) - 1; i >= 0; i-- {
		if err := cleanup[i](ctx); err != nil {
			shutdownErr = errors.Join(shutdownErr, err)
		}
	}
	return shutdownErr
}
