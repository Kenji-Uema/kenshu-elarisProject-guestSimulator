package flows

import (
	"testing"

	"github.com/Kenji-Uema/guestSimulator/internal/config"
	"github.com/Kenji-Uema/guestSimulator/internal/domain"
)

func TestNewPaymentFlowWithStateBuildsExpectedGraph(t *testing.T) {
	flow, err := NewPaymentFlowWithState(&domain.State{}, config.ServicesConfig{
		PaymentSimulatorUrl:  "payment-simulator",
		PaymentSimulatorPort: 8082,
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if flow == nil {
		t.Fatal("expected flow")
	}
	if flow.spanName != "GuestJourneyPaymentFlowSteps" {
		t.Fatalf("unexpected span name: %q", flow.spanName)
	}
	if flow.zeroStep.Name() != "GuestJourneyNoopStep" {
		t.Fatalf("unexpected zero step: %q", flow.zeroStep.Name())
	}
	if flow.firstStep.Name() != "PayInvoiceStep" {
		t.Fatalf("unexpected first step: %q", flow.firstStep.Name())
	}
	if len(flow.stateMap) != 1 {
		t.Fatalf("unexpected state map size: %d", len(flow.stateMap))
	}
}
