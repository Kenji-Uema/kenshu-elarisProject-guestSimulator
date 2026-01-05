package domain

import "time"

type Period struct {
	Start time.Time
	End   time.Time
}

type CottageAvailablePeriod struct {
	Name    string
	Periods []Period
}
