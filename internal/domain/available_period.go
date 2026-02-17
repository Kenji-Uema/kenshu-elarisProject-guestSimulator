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
