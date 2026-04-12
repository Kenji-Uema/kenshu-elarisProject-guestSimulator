package services

import (
	"math/rand"
	"testing"

	"github.com/Kenji-Uema/guestSimulator/internal/domain"
)

func TestPickRandomWeightedSkipsNonPositiveWeights(t *testing.T) {
	rand.Seed(1)

	got := PickRandomWeighted([]domain.WeightedTuple[string]{
		{Value: "ignored", Weight: -1},
		{Value: "selected", Weight: 1},
	})

	if got != "selected" {
		t.Fatalf("unexpected selection: %q", got)
	}
}

func TestPickRandomWeightedFallsBackToLastItemWhenAllWeightsAreNonPositive(t *testing.T) {
	rand.Seed(1)

	got := PickRandomWeighted([]domain.WeightedTuple[string]{
		{Value: "first", Weight: 0},
		{Value: "last", Weight: -1},
	})

	if got != "last" {
		t.Fatalf("unexpected selection: %q", got)
	}
}
