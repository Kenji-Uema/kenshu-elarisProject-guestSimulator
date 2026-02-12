package booking_state

import (
	"context"

	"github.com/Kenji-Uema/guestEmulator/internal/app/utils"
	"github.com/Kenji-Uema/guestEmulator/internal/domain"
)

type SelectCottagePeriodPreSetState struct{}

func NewSelectCottagePeriodPreSetState() *SelectCottagePeriodPreSetState {
	return &SelectCottagePeriodPreSetState{}
}

func (s SelectCottagePeriodPreSetState) Execute(_ context.Context, cottages []domain.CottageAvailable) (string, error) {
	return utils.PickRandom(cottages).CottageName, nil
}
