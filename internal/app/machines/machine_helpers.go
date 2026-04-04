package machines

import (
	"context"

	"github.com/Kenji-Uema/guestSimulator/internal/app/steps"
	"github.com/Kenji-Uema/guestSimulator/internal/domain"
)

type noopStep struct{}

func (noopStep) Name() string                  { return "GuestJourneyNoopStep" }
func (noopStep) Validate() error               { return nil }
func (noopStep) Execute(context.Context) error { return nil }

func replaceTransitionTarget(transitions []domain.WeightedTuple[steps.Step], oldTarget steps.Step, newTarget steps.Step) []domain.WeightedTuple[steps.Step] {
	replaced := make([]domain.WeightedTuple[steps.Step], 0, len(transitions))
	for _, transition := range transitions {
		value := transition.Value
		if value == oldTarget {
			value = newTarget
		}
		replaced = append(replaced, domain.WeightedTuple[steps.Step]{
			Value:  value,
			Weight: transition.Weight,
		})
	}
	return replaced
}

func removeTransitionByName(transitions []domain.WeightedTuple[steps.Step], targetName string) []domain.WeightedTuple[steps.Step] {
	filtered := make([]domain.WeightedTuple[steps.Step], 0, len(transitions))
	for _, transition := range transitions {
		if transition.Value != nil && transition.Value.Name() == targetName {
			continue
		}
		filtered = append(filtered, transition)
	}
	return filtered
}
