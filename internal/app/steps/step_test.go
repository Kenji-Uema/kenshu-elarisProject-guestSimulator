package steps

import (
	"testing"

	"github.com/Kenji-Uema/guestSimulator/internal/domain"
)

func TestStepInterfaceCanHoldConcreteSteps(t *testing.T) {
	var stepsList []Step
	stepsList = append(stepsList, NewNoopStep(), NewEndStep(&domain.State{}))

	if len(stepsList) != 2 {
		t.Fatalf("unexpected step count: %d", len(stepsList))
	}
	if stepsList[0].Name() == "" || stepsList[1].Name() == "" {
		t.Fatalf("expected named steps: %#v", stepsList)
	}
}
