package booking_step

import (
	"strings"
	"testing"

	"github.com/Kenji-Uema/guestSimulator/internal/domain"
	redisfakes "github.com/Kenji-Uema/guestSimulator/internal/infra/redis/fakes"
)

func TestNewSelectCottageStepReturnsNamedStep(t *testing.T) {
	step := NewSelectCottageStep(&domain.State{}, &redisfakes.Cache{})

	if step == nil {
		t.Fatal("expected step")
	}
	if step.Name() != "SelectCottageStep" {
		t.Fatalf("unexpected step name: %q", step.Name())
	}
}

func TestSelectCottageStepValidateRejectsEmptyCottageNames(t *testing.T) {
	err := SelectCottageStep{
		state: &domain.State{CottageNames: []string{}},
		cache: &redisfakes.Cache{},
	}.Validate()
	if err == nil || !strings.Contains(err.Error(), "cottageNames is empty") {
		t.Fatalf("unexpected error: %v", err)
	}
}
