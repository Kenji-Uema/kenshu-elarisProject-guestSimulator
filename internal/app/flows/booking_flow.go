package flows

import (
	"github.com/Kenji-Uema/guestSimulator/internal/app/steps"
	"github.com/Kenji-Uema/guestSimulator/internal/app/steps/booking_step"
	"github.com/Kenji-Uema/guestSimulator/internal/config"
	"github.com/Kenji-Uema/guestSimulator/internal/domain"
	"github.com/Kenji-Uema/guestSimulator/internal/infra/http"
	"github.com/Kenji-Uema/guestSimulator/internal/port"
)

func NewBookingFlow(state *domain.State, serviceConfig config.ServicesConfig, cache port.Cache, clock port.Clock) (*Flow, error) {
	cottageClient := http.NewRestyClient(serviceConfig.CottageManagerUrl, serviceConfig.CottageManagerPort)

	bookingSteps := config.BookingSteps{
		End:           steps.NewEndStep(state),
		ListCottages:  booking_step.NewListCottagesStep(state, cottageClient),
		SelectCottage: booking_step.NewSelectCottageStep(state, cache),
		SelectPeriod:  booking_step.NewSelectPeriodStep(state, clock, cottageClient, cache),
		BookCottage:   booking_step.NewBookCottageStep(state, cottageClient, cache),
	}

	flowDef := config.DefaultBookingFlow(bookingSteps)

	return &Flow{
		spanName:  "BookingFlow",
		zeroStep:  steps.NewNoopStep(),
		firstStep: flowDef.Start,
		stateMap:  flowDef.StateMap(),
	}, nil
}
