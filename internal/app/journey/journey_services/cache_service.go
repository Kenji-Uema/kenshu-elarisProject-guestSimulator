package journey_services

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Kenji-Uema/guestSimulator/internal/domain"
	"github.com/Kenji-Uema/guestSimulator/internal/domain/dto"
	"github.com/Kenji-Uema/guestSimulator/internal/port"
)

type JourneyCacheService struct {
	cache port.Cache
}

func NewJourneyCacheService(cache port.Cache) *JourneyCacheService {
	return &JourneyCacheService{cache: cache}
}

func (s *JourneyCacheService) InitializeStateCache(ctx context.Context, state *domain.State) error {
	if state == nil {
		return fmt.Errorf("invalid state, state is nil")
	}
	if s == nil || s.cache == nil {
		return fmt.Errorf("invalid guest journey cache")
	}
	if state.Guest == nil {
		return fmt.Errorf("invalid state, guest is nil")
	}
	if state.RedisKey == "" {
		return fmt.Errorf("invalid state, redisKey is empty")
	}

	if err := s.cache.Save(ctx, state, dto.GuestJourneyCacheValue{
		PersonalInfo: state.Guest,
	}); err != nil {
		return err
	}

	slog.InfoContext(ctx, "state cache initialized", "key", state.RedisKey, "guestEmail", state.Guest.Email)
	return nil
}

func (s *JourneyCacheService) SyncStateCache(ctx context.Context, state *domain.State, stepName string) error {
	if state == nil {
		return fmt.Errorf("invalid state, state is nil")
	}
	if s == nil || s.cache == nil {
		return fmt.Errorf("invalid guest journey cache")
	}
	if state.GuestId == "" {
		return fmt.Errorf("invalid state, guestId is empty")
	}

	current, err := s.cache.Load(ctx, state)
	if err != nil {
		return err
	}

	if state.GuestId != "" {
		current.GuestID = state.GuestId
	}
	if state.Guest != nil {
		current.PersonalInfo = state.Guest
	}

	if err := s.cache.Save(ctx, state, current); err != nil {
		return err
	}

	slog.InfoContext(ctx, "guest journey cache upserted", "key", state.RedisKey, "step", stepName)
	return nil
}

func (s *JourneyCacheService) LogStateCache(ctx context.Context, state *domain.State) error {
	if state == nil {
		return fmt.Errorf("invalid state, state is nil")
	}
	if s == nil || s.cache == nil {
		return fmt.Errorf("invalid guest journey cache")
	}
	if state.RedisKey == "" {
		return fmt.Errorf("invalid state, redisKey is empty")
	}

	value, err := s.cache.Get(ctx, state.RedisKey)
	if err != nil {
		return err
	}

	slog.InfoContext(ctx, "guest journey cache content", "key", state.RedisKey, "value", value)
	return nil
}

func (s *JourneyCacheService) DeleteStateCache(ctx context.Context, state *domain.State) error {
	if state == nil {
		return fmt.Errorf("invalid state, state is nil")
	}
	if s == nil || s.cache == nil {
		return fmt.Errorf("invalid guest journey cache")
	}
	if state.RedisKey == "" {
		return fmt.Errorf("invalid state, redisKey is empty")
	}

	if err := s.cache.Del(ctx, state.RedisKey); err != nil {
		return err
	}

	slog.InfoContext(ctx, "guest journey cache deleted", "key", state.RedisKey)
	return nil
}
