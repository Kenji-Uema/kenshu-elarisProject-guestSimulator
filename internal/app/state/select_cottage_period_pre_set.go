package state

import (
	"context"
	"guestEmulator/internal/app/utils"
	"guestEmulator/internal/domain"
)

type SelectCottagePeriodPreSetState struct{}

func NewSelectCottagePeriodPreSetState() *SelectCottagePeriodPreSetState {
	return &SelectCottagePeriodPreSetState{}
}

func (s SelectCottagePeriodPreSetState) Execute(_ context.Context, cottages []domain.CottageAvailable) (string, error) {
	return utils.PickRandom(cottages).CottageName, nil
}
