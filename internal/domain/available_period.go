package domain

import "time"

type Period struct {
	Start time.Time `json:"from"`
	End   time.Time `json:"to"`
}

type CottageAvailablePeriod struct {
	Name    string
	Periods []Period
}

type AvailablePeriodDTO struct {
	Name    string      `json:"cottage_name"`
	Periods []PeriodDTO `json:"available_periods"`
}

type PeriodDTO struct {
	CheckIn  time.Time `json:"check_in"`
	CheckOut time.Time `json:"check_out"`
}

func (d AvailablePeriodDTO) ToPeriods() []Period {
	periods := make([]Period, len(d.Periods))
	for i, period := range d.Periods {
		periods[i] = Period{
			Start: period.CheckIn,
			End:   period.CheckOut,
		}
	}

	return periods
}
