package redis

import (
	"context"
	"strings"
	"testing"

	"github.com/Kenji-Uema/guestSimulator/internal/domain"
	"github.com/Kenji-Uema/guestSimulator/internal/domain/dto"
)

func TestCacheEnsureKey(t *testing.T) {
	cache := NewCache(nil)

	state := &domain.State{GuestId: "guest-1"}
	key, err := cache.EnsureKey(state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if key != "guest.guest-1" || state.RedisKey != "guest.guest-1" {
		t.Fatalf("unexpected key: %q", key)
	}
}

func TestCacheEnsureKeyRejectsInvalidState(t *testing.T) {
	cache := NewCache(nil)

	if _, err := cache.EnsureKey(nil); err == nil || !strings.Contains(err.Error(), "state is nil") {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := cache.EnsureKey(&domain.State{}); err == nil || !strings.Contains(err.Error(), "guestId is empty") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCacheRejectsNilRedisClient(t *testing.T) {
	cache := NewCache(nil)

	if err := cache.Set(context.Background(), "key", "value"); err == nil {
		t.Fatal("expected set error")
	}
	if _, err := cache.Get(context.Background(), "key"); err == nil {
		t.Fatal("expected get error")
	}
	if err := cache.Del(context.Background(), "key"); err == nil {
		t.Fatal("expected del error")
	}
	if _, err := cache.Load(context.Background(), &domain.State{GuestId: "guest-1"}); err == nil {
		t.Fatal("expected load error")
	}
	if err := cache.Save(context.Background(), &domain.State{GuestId: "guest-1"}, dto.GuestJourneyCacheValue{}); err == nil {
		t.Fatal("expected save error")
	}
}
