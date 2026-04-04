package journeyctx

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Kenji-Uema/guestSimulator/internal/domain"
	"github.com/Kenji-Uema/guestSimulator/internal/domain/dto"
	redisc "github.com/Kenji-Uema/guestSimulator/internal/infra/redis"
)

func EnsureRedisKey(state *domain.State) (string, error) {
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

func Load(ctx context.Context, redis *redisc.Redis, state *domain.State) (dto.GuestJourneyCacheValue, error) {
	if redis == nil {
		return dto.GuestJourneyCacheValue{}, fmt.Errorf("invalid redis client")
	}

	key, err := EnsureRedisKey(state)
	if err != nil {
		return dto.GuestJourneyCacheValue{}, err
	}

	payload, err := redis.Get(ctx, key)
	if err != nil {
		return dto.GuestJourneyCacheValue{}, err
	}

	var value dto.GuestJourneyCacheValue
	if err := json.Unmarshal([]byte(payload), &value); err != nil {
		return dto.GuestJourneyCacheValue{}, err
	}

	return value, nil
}

func Save(ctx context.Context, redis *redisc.Redis, state *domain.State, value dto.GuestJourneyCacheValue) error {
	if redis == nil {
		return fmt.Errorf("invalid redis client")
	}

	key, err := EnsureRedisKey(state)
	if err != nil {
		return err
	}

	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return redis.Set(ctx, key, string(payload))
}
