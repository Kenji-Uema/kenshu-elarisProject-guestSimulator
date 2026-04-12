package booking_step

import (
	"strings"
	"testing"

	"github.com/Kenji-Uema/guestSimulator/internal/domain"
	"github.com/go-resty/resty/v2"
)

func TestNewListCottagesStepReturnsNamedStep(t *testing.T) {
	step := NewListCottagesStep(&domain.State{}, resty.New())

	if step == nil {
		t.Fatal("expected step")
	}
	if step.Name() != "ListCottagesStep" {
		t.Fatalf("unexpected step name: %q", step.Name())
	}
}

func TestListCottagesStepValidateRejectsNilState(t *testing.T) {
	err := ListCottagesStep{client: resty.New()}.Validate()
	if err == nil || !strings.Contains(err.Error(), "state is nil") {
		t.Fatalf("unexpected error: %v", err)
	}
}
