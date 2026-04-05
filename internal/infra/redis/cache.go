package redis

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Kenji-Uema/guestSimulator/internal/domain"
	"github.com/Kenji-Uema/guestSimulator/internal/domain/dto"
)

type Cache struct {
	redis *Redis
}

func NewCache(redis *Redis) *Cache {
	return &Cache{redis: redis}
}

func (c *Cache) Set(ctx context.Context, key string, value string) error {
	if c.redis == nil {
		return fmt.Errorf("invalid redis client")
	}

	return c.redis.set(ctx, key, value)
}

func (c *Cache) Get(ctx context.Context, key string) (string, error) {
	if c.redis == nil {
		return "", fmt.Errorf("invalid redis client")
	}

	return c.redis.get(ctx, key)
}

func (c *Cache) Del(ctx context.Context, key string) error {
	if c.redis == nil {
		return fmt.Errorf("invalid redis client")
	}

	return c.redis.del(ctx, key)
}

func (c *Cache) EnsureKey(state *domain.State) (string, error) {
	if state == nil {
		return "", fmt.Errorf("invalid state, state is nil")
	}
	if state.RedisKey != "" {
		return state.RedisKey, nil
	}
	if state.GuestId == "" {
		return "", fmt.Errorf("invalid state, guestId is empty")
	}

	state.RedisKey = fmt.Sprintf("guest.%s", state.GuestId)
	return state.RedisKey, nil
}

func (c *Cache) Load(ctx context.Context, state *domain.State) (dto.GuestJourneyCacheValue, error) {
	if c.redis == nil {
		return dto.GuestJourneyCacheValue{}, fmt.Errorf("invalid redis client")
	}

	key, err := c.EnsureKey(state)
	if err != nil {
		return dto.GuestJourneyCacheValue{}, err
	}

	payload, err := c.Get(ctx, key)
	if err != nil {
		return dto.GuestJourneyCacheValue{}, err
	}

	var value dto.GuestJourneyCacheValue
	if err := json.Unmarshal([]byte(payload), &value); err != nil {
		return dto.GuestJourneyCacheValue{}, err
	}

	return value, nil
}

func (c *Cache) Save(ctx context.Context, state *domain.State, value dto.GuestJourneyCacheValue) error {
	if c.redis == nil {
		return fmt.Errorf("invalid redis client")
	}

	key, err := c.EnsureKey(state)
	if err != nil {
		return err
	}

	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return c.Set(ctx, key, string(payload))
}
