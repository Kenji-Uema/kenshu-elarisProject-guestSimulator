package redis

import (
	"context"

	goredis "github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type Redis struct {
	client *goredis.Client
}

var tracer = otel.Tracer("guest-simulator.redis")

func NewRedisClient(client *goredis.Client) *Redis {
	return &Redis{client: client}
}

func (r *Redis) Ping(ctx context.Context) error {
	ctx, span := tracer.Start(ctx, "redis PING")
	defer span.End()

	span.SetAttributes(
		attribute.String("db.system", "redis"),
		attribute.String("db.operation", "PING"),
	)

	err := r.client.Ping(ctx).Err()
	recordSpanError(span, err)
	return err
}

func (r *Redis) Set(ctx context.Context, key string, value string) error {
	ctx, span := tracer.Start(ctx, "redis SET")
	defer span.End()

	span.SetAttributes(
		attribute.String("db.system", "redis"),
		attribute.String("db.operation", "SET"),
		attribute.String("db.redis.key", key),
		attribute.Int("db.redis.value_size", len(value)),
	)

	err := r.client.Set(ctx, key, value, 0).Err()
	recordSpanError(span, err)
	return err
}

func (r *Redis) Get(ctx context.Context, key string) (string, error) {
	ctx, span := tracer.Start(ctx, "redis GET")
	defer span.End()

	span.SetAttributes(
		attribute.String("db.system", "redis"),
		attribute.String("db.operation", "GET"),
		attribute.String("db.redis.key", key),
	)

	value, err := r.client.Get(ctx, key).Result()
	recordSpanError(span, err)
	return value, err
}

func (r *Redis) Del(ctx context.Context, key string) error {
	ctx, span := tracer.Start(ctx, "redis DEL")
	defer span.End()

	span.SetAttributes(
		attribute.String("db.system", "redis"),
		attribute.String("db.operation", "DEL"),
		attribute.String("db.redis.key", key),
	)

	err := r.client.Del(ctx, key).Err()
	recordSpanError(span, err)
	return err
}

func (r *Redis) Close() error {
	return r.client.Close()
}

func recordSpanError(span trace.Span, err error) {
	if err == nil {
		return
	}

	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
}
