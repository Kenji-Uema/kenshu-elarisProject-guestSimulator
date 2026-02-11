package booking_state

import (
	"context"
	"fmt"
	"math/rand"
)

type SelectCottageState struct{}

func NewSelectCottageState() *SelectCottageState {
	return &SelectCottageState{}
}

func (s *SelectCottageState) Execute(_ context.Context, cottages []string) (string, error) {
	if cottages == nil || len(cottages) == 0 {
		return "", fmt.Errorf("input invalid; input=%v", cottages)
	}

	selected := cottages[rand.Intn(len(cottages))]

	return selected, nil
}
