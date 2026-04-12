package flows

import (
	"testing"

	"github.com/Kenji-Uema/guestSimulator/internal/config"
	"github.com/Kenji-Uema/guestSimulator/internal/domain"
)

func TestNewGuestRegisterFlowWithStateBuildsExpectedGraph(t *testing.T) {
	flow, err := NewGuestRegisterFlowWithState(&domain.State{}, config.ServicesConfig{
		GuestManagerUrl:  "guest-manager",
		GuestManagerPort: 8081,
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if flow == nil {
		t.Fatal("expected flow")
	}
	if flow.spanName != "GuestJourneyGuestRegisterFlow" {
		t.Fatalf("unexpected span name: %q", flow.spanName)
	}
	if flow.zeroStep.Name() != "GuestJourneyNoopStep" {
		t.Fatalf("unexpected zero step: %q", flow.zeroStep.Name())
	}
	if flow.firstStep.Name() != "RegisterGuestStep" {
		t.Fatalf("unexpected first step: %q", flow.firstStep.Name())
	}
	if len(flow.stateMap) != 2 {
		t.Fatalf("unexpected state map size: %d", len(flow.stateMap))
	}
}
