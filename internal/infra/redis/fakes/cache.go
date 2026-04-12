package fakes

import (
	"context"
	"fmt"

	"github.com/Kenji-Uema/guestSimulator/internal/domain"
	"github.com/Kenji-Uema/guestSimulator/internal/domain/dto"
)

type Cache struct {
	SetErr       error
	GetValue     string
	GetErr       error
	DelErr       error
	EnsureKeyErr error
	LoadValue    dto.GuestJourneyCacheValue
	LoadErr      error
	SaveErr      error
	SavedValue   dto.GuestJourneyCacheValue
	DeletedKey   string
}

func (c *Cache) Set(context.Context, string, string) error {
	return c.SetErr
}

func (c *Cache) Get(context.Context, string) (string, error) {
	return c.GetValue, c.GetErr
}

func (c *Cache) Del(_ context.Context, key string) error {
	c.DeletedKey = key
	return c.DelErr
}

func (c *Cache) EnsureKey(state *domain.State) (string, error) {
	if c.EnsureKeyErr != nil {
		return "", c.EnsureKeyErr
	}
	if state == nil {
		return "", fmt.Errorf("nil state")
	}
	if state.RedisKey != "" {
		return state.RedisKey, nil
	}
	if state.GuestId == "" {
		return "", fmt.Errorf("empty guest id")
	}

	state.RedisKey = "guest." + state.GuestId
	return state.RedisKey, nil
}

func (c *Cache) Load(context.Context, *domain.State) (dto.GuestJourneyCacheValue, error) {
	return c.LoadValue, c.LoadErr
}

func (c *Cache) Save(_ context.Context, _ *domain.State, value dto.GuestJourneyCacheValue) error {
	c.SavedValue = value
	c.LoadValue = value
	return c.SaveErr
}
