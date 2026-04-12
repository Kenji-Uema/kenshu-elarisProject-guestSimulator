package flows

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/Kenji-Uema/guestSimulator/internal/app/steps"
	"github.com/Kenji-Uema/guestSimulator/internal/domain"
)

type recordingStep struct {
	name        string
	order       *[]string
	validateErr error
	executeErr  error
}

func (s recordingStep) Validate() error {
	if s.order != nil {
		*s.order = append(*s.order, "validate:"+s.name)
	}
	return s.validateErr
}

func (s recordingStep) Execute(context.Context) error {
	if s.order != nil {
		*s.order = append(*s.order, "execute:"+s.name)
	}
	return s.executeErr
}

func (s recordingStep) Name() string {
	return s.name
}

func TestRunStepGraphExecutesStepsInOrder(t *testing.T) {
	var order []string
	zero := recordingStep{name: "zero", order: &order}
	first := recordingStep{name: "first", order: &order}
	second := recordingStep{name: "second", order: &order}

	err := runStepGraph(context.Background(), "test-flow", zero, first, map[steps.Step][]domain.WeightedTuple[steps.Step]{
		first:  {{Value: second, Weight: 1}},
		second: nil,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []string{
		"execute:zero",
		"validate:first",
		"execute:first",
		"validate:second",
		"execute:second",
	}
	if !reflect.DeepEqual(order, want) {
		t.Fatalf("unexpected execution order: %#v", order)
	}
}

func TestRunStepGraphReturnsValidationError(t *testing.T) {
	var order []string
	zero := recordingStep{name: "zero", order: &order}
	invalid := recordingStep{name: "invalid", order: &order, validateErr: errors.New("boom")}

	err := runStepGraph(context.Background(), "test-flow", zero, invalid, map[steps.Step][]domain.WeightedTuple[steps.Step]{
		invalid: nil,
	})
	if err == nil || err.Error() != "boom" {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []string{"execute:zero", "validate:invalid"}
	if !reflect.DeepEqual(order, want) {
		t.Fatalf("unexpected execution order: %#v", order)
	}
}

func TestRunLodgingStayFlowRejectsNilRunStep(t *testing.T) {
	err := RunLodgingStayFlow(context.Background(), &LodgingFlow{})
	if err == nil {
		t.Fatal("expected error for missing run step")
	}
}

func TestRunLodgingStayFlowRunsConfiguredStep(t *testing.T) {
	var order []string
	run := recordingStep{name: "run-lodging", order: &order}

	err := RunLodgingStayFlow(context.Background(), &LodgingFlow{RunLodgingStep: run})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []string{"validate:run-lodging", "execute:run-lodging"}
	if !reflect.DeepEqual(order, want) {
		t.Fatalf("unexpected execution order: %#v", order)
	}
}

func TestFlowStartUsesConfiguredGraph(t *testing.T) {
	var order []string
	flow := &Flow{
		spanName:  "test-flow",
		zeroStep:  recordingStep{name: "zero", order: &order},
		firstStep: recordingStep{name: "first", order: &order},
		stateMap: map[steps.Step][]domain.WeightedTuple[steps.Step]{
			recordingStep{name: "first", order: &order}: nil,
		},
	}

	if err := flow.Start(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(order) == 0 {
		t.Fatal("expected flow to execute steps")
	}
}
