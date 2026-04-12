package steps

import (
	"context"
	"testing"

	"github.com/Kenji-Uema/guestSimulator/internal/domain"
)

func TestNoopStep(t *testing.T) {
	step := NewNoopStep()

	if step.Name() != "GuestJourneyNoopStep" {
		t.Fatalf("unexpected step name: %q", step.Name())
	}
	if err := step.Validate(); err != nil {
		t.Fatalf("unexpected validate error: %v", err)
	}
	if err := step.Execute(context.Background()); err != nil {
		t.Fatalf("unexpected execute error: %v", err)
	}
}

func TestEndStep(t *testing.T) {
	step := NewEndStep(&domain.State{GuestId: "guest-1"})

	if step.Name() != "EndStep" {
		t.Fatalf("unexpected step name: %q", step.Name())
	}
	if err := step.Validate(); err != nil {
		t.Fatalf("unexpected validate error: %v", err)
	}
	if err := step.Execute(context.Background()); err != nil {
		t.Fatalf("unexpected execute error: %v", err)
	}
}
