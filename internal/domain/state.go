package domain

type State struct {
	Guest           *Guest
	GuestId         string
	CottageNames    []string
	SelectedCottage string
	SelectedPeriod  *Period
	BookingId       string
}
