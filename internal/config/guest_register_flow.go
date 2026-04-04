package config

import (
	"github.com/Kenji-Uema/guestSimulator/internal/app/steps"
	"github.com/Kenji-Uema/guestSimulator/internal/domain"
)

type GuestRegisterFlowSteps struct {
	Start         steps.Step
	RegisterGuest steps.Step
	RetrieveGuest steps.Step
	End           steps.Step
}

type GuestRegisterTransitions struct {
	RegisterGuestTransitions []domain.WeightedTuple[steps.Step]
	RetrieveGuestTransitions []domain.WeightedTuple[steps.Step]
}

type GuestRegisterFlow struct {
	GuestRegisterFlowSteps
	GuestRegisterTransitions
}

func DefaultGuestRegisterFlow(flow GuestRegisterFlowSteps) GuestRegisterFlow {
	return GuestRegisterFlow{
		GuestRegisterFlowSteps: GuestRegisterFlowSteps{
			Start:         flow.RegisterGuest,
			RegisterGuest: flow.RegisterGuest,
			RetrieveGuest: flow.RetrieveGuest,
			End:           flow.End,
		},
		GuestRegisterTransitions: GuestRegisterTransitions{
			RegisterGuestTransitions: []domain.WeightedTuple[steps.Step]{
				{Value: flow.RetrieveGuest, Weight: 1.0},
			},
			RetrieveGuestTransitions: []domain.WeightedTuple[steps.Step]{
				{Value: flow.End, Weight: 1.0},
			},
		},
	}
}

func (f GuestRegisterFlow) StateMap() map[steps.Step][]domain.WeightedTuple[steps.Step] {
	return map[steps.Step][]domain.WeightedTuple[steps.Step]{
		f.RegisterGuest: f.RegisterGuestTransitions,
		f.RetrieveGuest: f.RetrieveGuestTransitions,
	}
}
