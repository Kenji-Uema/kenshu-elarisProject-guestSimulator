package config

import (
	"github.com/Kenji-Uema/guestSimulator/internal/app/steps"
	"github.com/Kenji-Uema/guestSimulator/internal/domain"
)

type PaymentSteps struct {
	Start      steps.Step
	PayInvoice steps.Step
	End        steps.Step
}

type PaymentTransitions struct {
	PayInvoiceTransitions []domain.WeightedTuple[steps.Step]
}

type PaymentFlow struct {
	PaymentSteps
	PaymentTransitions
}

func DefaultPaymentFlow(flow PaymentSteps) PaymentFlow {
	return PaymentFlow{
		PaymentSteps: PaymentSteps{
			Start:      flow.PayInvoice,
			PayInvoice: flow.PayInvoice,
			End:        flow.End,
		},
		PaymentTransitions: PaymentTransitions{
			PayInvoiceTransitions: []domain.WeightedTuple[steps.Step]{
				{Value: flow.End, Weight: 1.0},
			},
		},
	}
}

func (f PaymentFlow) StateMap() map[steps.Step][]domain.WeightedTuple[steps.Step] {
	return map[steps.Step][]domain.WeightedTuple[steps.Step]{
		f.PayInvoice: f.PayInvoiceTransitions,
	}
}
