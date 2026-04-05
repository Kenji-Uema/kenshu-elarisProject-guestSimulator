package flows

import (
	"github.com/Kenji-Uema/guestSimulator/internal/app/steps"
	"github.com/Kenji-Uema/guestSimulator/internal/app/steps/payment_step"
	"github.com/Kenji-Uema/guestSimulator/internal/config"
	"github.com/Kenji-Uema/guestSimulator/internal/domain"
	"github.com/Kenji-Uema/guestSimulator/internal/infra/http"
	"github.com/Kenji-Uema/guestSimulator/internal/port"
)

func NewPaymentFlowWithState(state *domain.State, serviceConfig config.ServicesConfig, cache port.Cache) (*Flow, error) {
	paymentClient := http.NewRestyClient(serviceConfig.PaymentSimulatorUrl, serviceConfig.PaymentSimulatorPort)

	payInvoiceStep := payment_step.NewPayInvoiceStep(state, paymentClient, cache)
	endStep := steps.NewEndStep(state)

	flow := config.DefaultPaymentFlow(config.PaymentSteps{
		PayInvoice: payInvoiceStep,
		End:        endStep,
	})

	return &Flow{
		spanName:  "PaymentFlow",
		zeroStep:  steps.NewNoopStep(),
		firstStep: flow.Start,
		stateMap:  flow.StateMap(),
	}, nil
}
