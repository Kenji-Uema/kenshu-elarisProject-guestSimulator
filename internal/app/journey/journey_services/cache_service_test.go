package journey_services

import (
	"context"
	"strings"
	"testing"

	"github.com/Kenji-Uema/guestSimulator/internal/domain"
	"github.com/Kenji-Uema/guestSimulator/internal/domain/dto/guest_registration"
	redisfakes "github.com/Kenji-Uema/guestSimulator/internal/infra/redis/fakes"
)

func TestNewJourneyCacheServiceReturnsService(t *testing.T) {
	cache := &redisfakes.Cache{}
	service := NewJourneyCacheService(cache)

	if service == nil || service.cache != cache {
		t.Fatalf("unexpected cache service: %#v", service)
	}
}

func TestJourneyCacheServiceInitializeStateCacheRejectsInvalidState(t *testing.T) {
	service := NewJourneyCacheService(&redisfakes.Cache{})

	if err := service.InitializeStateCache(context.Background(), nil); err == nil || !strings.Contains(err.Error(), "state is nil") {
		t.Fatalf("unexpected nil-state error: %v", err)
	}

	err := service.InitializeStateCache(context.Background(), &domain.State{
		Guest: &guest_registration.Guest{Email: "guest@test.com"},
	})
	if err == nil || !strings.Contains(err.Error(), "redisKey is empty") {
		t.Fatalf("unexpected missing-redis-key error: %v", err)
	}
}
