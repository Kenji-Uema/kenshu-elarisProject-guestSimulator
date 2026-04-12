package steps

import "testing"

func TestNewNoopStepReturnsNoopStep(t *testing.T) {
	step := NewNoopStep()

	if step == nil {
		t.Fatal("expected noop step")
	}
	if step.Name() != "GuestJourneyNoopStep" {
		t.Fatalf("unexpected step name: %q", step.Name())
	}
}
