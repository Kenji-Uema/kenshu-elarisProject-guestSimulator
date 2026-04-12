package booking

import (
	"testing"
	"time"
)

func TestAvailablePeriodDTOToPeriods(t *testing.T) {
	checkIn := time.Date(2026, time.April, 12, 0, 0, 0, 0, time.UTC)
	checkOut := checkIn.AddDate(0, 0, 3)

	dto := AvailablePeriodDTO{
		Name: "Alps",
		Periods: []PeriodDTO{{
			CheckIn:  checkIn,
			CheckOut: checkOut,
		}},
	}

	periods := dto.ToPeriods()
	if len(periods) != 1 {
		t.Fatalf("unexpected periods length: %d", len(periods))
	}
	if !periods[0].Start.Equal(checkIn) || !periods[0].End.Equal(checkOut) {
		t.Fatalf("unexpected periods: %#v", periods)
	}
}
