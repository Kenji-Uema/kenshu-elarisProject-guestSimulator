package journey_services

import (
	"context"
	"testing"

	"github.com/Kenji-Uema/guestSimulator/internal/domain"
)

func TestCloseCommunicationNoConsumerIsNoop(t *testing.T) {
	state := &domain.State{QueueName: "queue", RoutingKey: "routing"}

	if err := CloseCommunication(context.Background(), state, nil); err != nil {
		t.Fatalf("unexpected nil-bus error: %v", err)
	}
	if err := CloseCommunication(context.Background(), state, &GuestCommunicationBus{}); err != nil {
		t.Fatalf("unexpected empty-bus error: %v", err)
	}
}
