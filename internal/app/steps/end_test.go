package steps

import (
	"testing"

	"github.com/Kenji-Uema/guestSimulator/internal/domain"
)

func TestNewEndStepReturnsEndStep(t *testing.T) {
	step := NewEndStep(&domain.State{})

	if step == nil {
		t.Fatal("expected end step")
	}
	if step.Name() != "EndStep" {
		t.Fatalf("unexpected step name: %q", step.Name())
	}
}
