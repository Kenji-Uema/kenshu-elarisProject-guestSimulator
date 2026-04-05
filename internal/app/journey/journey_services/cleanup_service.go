package journey_services

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Kenji-Uema/guestSimulator/internal/domain"
	"go.opentelemetry.io/otel/attribute"
)

func CleanupState(ctx context.Context, state *domain.State, cacheService *JourneyCacheService, bus *GuestCommunicationBus) error {
	if err := cacheService.DeleteStateCache(ctx, state); err != nil {
		return err
	}

	return CloseCommunication(ctx, state, bus)
}

func CloseCommunication(ctx context.Context, state *domain.State, bus *GuestCommunicationBus) error {
	if state == nil {
		return fmt.Errorf("invalid state, state is nil")
	}
	if bus == nil || bus.Consumer == nil {
		return nil
	}

	if err := bus.Consumer.CloseChannel(); err != nil {
		return err
	}

	bus.Consumer = nil
	bus.Reset()
	slog.InfoContext(ctx, "guest communication queue closed", "queue", state.QueueName, "routingKey", state.RoutingKey)
	return nil
}

func CommunicationCleanupAttributes(state *domain.State) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String("guest.queue.name", state.QueueName),
		attribute.String("guest.queue.routing_key", state.RoutingKey),
		attribute.Bool("guest.queue.auto_delete", true),
	}
}
