package port

import (
	"context"

	"github.com/Kenji-Uema/guestSimulator/internal/domain"
	"github.com/Kenji-Uema/guestSimulator/internal/domain/dto"
)

type Cache interface {
	Set(ctx context.Context, key string, value string) error
	Get(ctx context.Context, key string) (string, error)
	Del(ctx context.Context, key string) error
	EnsureKey(state *domain.State) (string, error)
	Load(ctx context.Context, state *domain.State) (dto.GuestJourneyCacheValue, error)
	Save(ctx context.Context, state *domain.State, value dto.GuestJourneyCacheValue) error
}
