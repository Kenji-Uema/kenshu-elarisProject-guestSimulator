package flows

import (
	"testing"

	"github.com/Kenji-Uema/guestSimulator/internal/config"
	"github.com/Kenji-Uema/guestSimulator/internal/domain"
)

func TestNewBookingFlowBuildsExpectedGraph(t *testing.T) {
	flow, err := NewBookingFlow(&domain.State{}, config.ServicesConfig{
		CottageManagerUrl:  "cottage-manager",
		CottageManagerPort: 8080,
	}, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if flow == nil {
		t.Fatal("expected flow")
	}
	if flow.spanName != "GuestJourneyBookingFlow" {
		t.Fatalf("unexpected span name: %q", flow.spanName)
	}
	if flow.zeroStep == nil || flow.firstStep == nil {
		t.Fatalf("expected zero and first step: %#v", flow)
	}
	if flow.zeroStep.Name() != "GuestJourneyNoopStep" {
		t.Fatalf("unexpected zero step: %q", flow.zeroStep.Name())
	}
	if flow.firstStep.Name() != "ListCottagesStep" {
		t.Fatalf("unexpected first step: %q", flow.firstStep.Name())
	}
	if len(flow.stateMap) != 4 {
		t.Fatalf("unexpected state map size: %d", len(flow.stateMap))
	}
}
