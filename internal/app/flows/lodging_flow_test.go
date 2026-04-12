package flows

import (
	"testing"

	"github.com/Kenji-Uema/guestSimulator/internal/config"
	"github.com/Kenji-Uema/guestSimulator/internal/domain"
)

func TestNewLodgingFlowWithStateBuildsExpectedGraph(t *testing.T) {
	flow, err := NewLodgingFlowWithState(&domain.State{}, config.ServicesConfig{
		GuestManagerUrl:  "guest-manager",
		GuestManagerPort: 8081,
	}, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if flow == nil || flow.Flow == nil {
		t.Fatalf("expected lodging flow: %#v", flow)
	}
	if flow.spanName != "GuestJourneyLodgingFlow" {
		t.Fatalf("unexpected span name: %q", flow.spanName)
	}
	if flow.zeroStep.Name() != "GuestJourneyNoopStep" {
		t.Fatalf("unexpected zero step: %q", flow.zeroStep.Name())
	}
	if flow.RunLodgingStep == nil || flow.firstStep != flow.RunLodgingStep {
		t.Fatalf("unexpected run step wiring: %#v", flow)
	}
	if len(flow.stateMap) != 1 {
		t.Fatalf("unexpected state map size: %d", len(flow.stateMap))
	}
}
