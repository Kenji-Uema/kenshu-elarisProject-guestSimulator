package config

import (
	"github.com/Kenji-Uema/guestSimulator/internal/app/steps"
	"github.com/Kenji-Uema/guestSimulator/internal/domain"
)

type BookingSteps struct {
	Start         steps.Step
	ListCottages  steps.Step
	SelectCottage steps.Step
	SelectPeriod  steps.Step
	RegisterGuest steps.Step
	BookCottage   steps.Step
	End           steps.Step
}

type BookingTransitions struct {
	ListCottagesTransitions  []domain.WeightedTuple[steps.Step]
	SelectCottageTransitions []domain.WeightedTuple[steps.Step]
	SelectPeriodTransitions  []domain.WeightedTuple[steps.Step]
	RegisterGuestTransitions []domain.WeightedTuple[steps.Step]
	BookCottageTransitions   []domain.WeightedTuple[steps.Step]
}

type BookingFlow struct {
	BookingSteps
	BookingTransitions
}

func DefaultBookingFlow(flow BookingSteps) BookingFlow {
	return BookingFlow{
		BookingSteps: BookingSteps{
			Start:         flow.ListCottages,
			ListCottages:  flow.ListCottages,
			SelectCottage: flow.SelectCottage,
			SelectPeriod:  flow.SelectPeriod,
			RegisterGuest: flow.RegisterGuest,
			BookCottage:   flow.BookCottage,
			End:           flow.End,
		},
		BookingTransitions: BookingTransitions{
			ListCottagesTransitions: []domain.WeightedTuple[steps.Step]{
				{Value: flow.SelectCottage, Weight: 0.80},
				{Value: flow.End, Weight: 0.20},
			},
			SelectCottageTransitions: []domain.WeightedTuple[steps.Step]{
				{Value: flow.SelectPeriod, Weight: 0.60},
				{Value: flow.ListCottages, Weight: 0.30},
				{Value: flow.End, Weight: 0.10},
			},
			SelectPeriodTransitions: []domain.WeightedTuple[steps.Step]{
				{Value: flow.SelectPeriod, Weight: 0.30},
				{Value: flow.SelectCottage, Weight: 0.20},
				{Value: flow.ListCottages, Weight: 0.05},
				{Value: flow.RegisterGuest, Weight: 0.40},
				{Value: flow.End, Weight: 0.05},
			},
			RegisterGuestTransitions: []domain.WeightedTuple[steps.Step]{
				{Value: flow.BookCottage, Weight: 0.90},
				{Value: flow.End, Weight: 0.10},
			},
			BookCottageTransitions: []domain.WeightedTuple[steps.Step]{
				{Value: flow.SelectPeriod, Weight: 0.05},
				{Value: flow.SelectCottage, Weight: 0.05},
				{Value: flow.ListCottages, Weight: 0.05},
				{Value: flow.End, Weight: 0.85},
			},
		},
	}
}

func BookingFlowWithoutGuestRegistration(flow BookingSteps) BookingFlow {
	return BookingFlow{
		BookingSteps: BookingSteps{
			Start:         flow.ListCottages,
			ListCottages:  flow.ListCottages,
			SelectCottage: flow.SelectCottage,
			SelectPeriod:  flow.SelectPeriod,
			RegisterGuest: flow.RegisterGuest,
			BookCottage:   flow.BookCottage,
			End:           flow.End,
		},
		BookingTransitions: BookingTransitions{
			ListCottagesTransitions: []domain.WeightedTuple[steps.Step]{
				{Value: flow.SelectCottage, Weight: 0.80},
				{Value: flow.End, Weight: 0.20},
			},
			SelectCottageTransitions: []domain.WeightedTuple[steps.Step]{
				{Value: flow.SelectPeriod, Weight: 0.60},
				{Value: flow.ListCottages, Weight: 0.30},
				{Value: flow.End, Weight: 0.10},
			},
			SelectPeriodTransitions: []domain.WeightedTuple[steps.Step]{
				{Value: flow.SelectPeriod, Weight: 0.30},
				{Value: flow.SelectCottage, Weight: 0.20},
				{Value: flow.ListCottages, Weight: 0.05},
				{Value: flow.BookCottage, Weight: 0.40},
				{Value: flow.End, Weight: 0.05},
			},
			RegisterGuestTransitions: []domain.WeightedTuple[steps.Step]{
				{Value: flow.BookCottage, Weight: 1.0},
			},
			BookCottageTransitions: []domain.WeightedTuple[steps.Step]{
				{Value: flow.SelectPeriod, Weight: 0.05},
				{Value: flow.SelectCottage, Weight: 0.05},
				{Value: flow.ListCottages, Weight: 0.05},
				{Value: flow.End, Weight: 0.85},
			},
		},
	}
}

func (f BookingFlow) StateMap() map[steps.Step][]domain.WeightedTuple[steps.Step] {
	return map[steps.Step][]domain.WeightedTuple[steps.Step]{
		f.ListCottages:  f.ListCottagesTransitions,
		f.SelectCottage: f.SelectCottageTransitions,
		f.SelectPeriod:  f.SelectPeriodTransitions,
		f.RegisterGuest: f.RegisterGuestTransitions,
		f.BookCottage:   f.BookCottageTransitions,
	}
}
