package config

import (
	"github.com/Kenji-Uema/guestSimulator/internal/app/steps"
	"github.com/Kenji-Uema/guestSimulator/internal/domain"
)

type PaymentSteps struct {
	Start                   steps.Step
	WaitForInvoice          steps.Step
	PayInvoice              steps.Step
	WaitForConfirmedBooking steps.Step
	End                     steps.Step
}

type PaymentTransitions struct {
	WaitForInvoiceTransitions          []domain.WeightedTuple[steps.Step]
	PayInvoiceTransitions              []domain.WeightedTuple[steps.Step]
	WaitForConfirmedBookingTransitions []domain.WeightedTuple[steps.Step]
}

type PaymentFlow struct {
	PaymentSteps
	PaymentTransitions
}

func DefaultPaymentFlow(flow PaymentSteps) PaymentFlow {
	return PaymentFlow{
		PaymentSteps: PaymentSteps{
			Start:                   flow.WaitForInvoice,
			WaitForInvoice:          flow.WaitForInvoice,
			PayInvoice:              flow.PayInvoice,
			WaitForConfirmedBooking: flow.WaitForConfirmedBooking,
			End:                     flow.End,
		},
		PaymentTransitions: PaymentTransitions{
			WaitForInvoiceTransitions: []domain.WeightedTuple[steps.Step]{
				{Value: flow.PayInvoice, Weight: 1.0},
			},
			PayInvoiceTransitions: []domain.WeightedTuple[steps.Step]{
				{Value: flow.WaitForConfirmedBooking, Weight: 1.0},
			},
			WaitForConfirmedBookingTransitions: []domain.WeightedTuple[steps.Step]{
				{Value: flow.End, Weight: 1.0},
			},
		},
	}
}

func (f PaymentFlow) StateMap() map[steps.Step][]domain.WeightedTuple[steps.Step] {
	return map[steps.Step][]domain.WeightedTuple[steps.Step]{
		f.WaitForInvoice:          f.WaitForInvoiceTransitions,
		f.PayInvoice:              f.PayInvoiceTransitions,
		f.WaitForConfirmedBooking: f.WaitForConfirmedBookingTransitions,
	}
}
