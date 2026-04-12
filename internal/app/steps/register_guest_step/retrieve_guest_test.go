package register_guest_step

import (
	"strings"
	"testing"

	"github.com/Kenji-Uema/guestSimulator/internal/domain"
	"github.com/go-resty/resty/v2"
)

func TestNewRetrieveGuestStepReturnsNamedStep(t *testing.T) {
	step := NewRetrieveGuestStep(resty.New(), &domain.State{})

	if step == nil {
		t.Fatal("expected step")
	}
	if step.Name() != "RetrieveGuestStep" {
		t.Fatalf("unexpected step name: %q", step.Name())
	}
}

func TestRetrieveGuestStepValidateRejectsMissingGuestID(t *testing.T) {
	err := RetrieveGuestStep{state: &domain.State{}}.Validate()
	if err == nil || !strings.Contains(err.Error(), "guestId is empty") {
		t.Fatalf("unexpected error: %v", err)
	}
}
