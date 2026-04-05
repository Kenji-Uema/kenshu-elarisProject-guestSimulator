package flows

import (
	"context"
	"fmt"

	"github.com/Kenji-Uema/guestSimulator/internal/app/steps"
	"github.com/Kenji-Uema/guestSimulator/internal/app/steps/payment_step"
	"github.com/Kenji-Uema/guestSimulator/internal/config"
	"github.com/Kenji-Uema/guestSimulator/internal/domain"
	"github.com/Kenji-Uema/guestSimulator/internal/infra/http"
	"github.com/Kenji-Uema/guestSimulator/internal/port"
)

func NewPaymentFlowWithState(state *domain.State, flowConfig config.PaymentFlowConfig, serviceConfig config.ServicesConfig, cache port.Cache) (*Flow, error) {
	guestClient := http.NewRestyClient(serviceConfig.GuestManagerUrl, serviceConfig.GuestManagerPort)
	paymentClient := http.NewRestyClient(serviceConfig.PaymentSimulatorUrl, serviceConfig.PaymentSimulatorPort)

	waitForInvoiceStep := payment_step.NewWaitForInvoiceStep(state, guestClient, paymentClient, cache)
	payInvoiceStep := payment_step.NewPayInvoiceStep(state, guestClient, paymentClient, cache)
	waitForConfirmedBookingStep := payment_step.NewWaitForConfirmedBookingStep(state, guestClient, paymentClient, cache)
	endStep := steps.NewEndStep(state)

	flow := config.DefaultPaymentFlow(config.PaymentSteps{
		WaitForInvoice:          waitForInvoiceStep,
		PayInvoice:              payInvoiceStep,
		WaitForConfirmedBooking: waitForConfirmedBookingStep,
		End:                     endStep,
	})

	return &Flow{
		spanName:                  "PaymentFlow",
		zeroStep:                  steps.NewInitStep(state),
		firstStep:                 flow.Start,
		stateMap:                  flow.StateMap(),
		timeBetweenStepsInSeconds: flowConfig.TimeBetweenStepsInSeconds,
	}, nil
}

func RunPaymentFlow(ctx context.Context, flow *Flow, timeBetweenStepsInSeconds int) error {
	if flow == nil {
		return fmt.Errorf("failed to compose payment journey from payment flow")
	}

	var waitForInvoiceStep steps.Step
	var payInvoiceStep steps.Step
	var waitForConfirmedBookingStep steps.Step

	if flow.firstStep != nil && flow.firstStep.Name() == "WaitForInvoiceStep" {
		waitForInvoiceStep = flow.firstStep
	}

	for step := range flow.stateMap {
		if step == nil {
			continue
		}
		switch step.Name() {
		case "WaitForInvoiceStep":
			waitForInvoiceStep = step
		case "PayInvoiceStep":
			payInvoiceStep = step
		case "WaitForConfirmedBookingStep":
			waitForConfirmedBookingStep = step
		}
	}

	if waitForInvoiceStep == nil || payInvoiceStep == nil || waitForConfirmedBookingStep == nil {
		return fmt.Errorf("failed to compose payment journey from payment flow")
	}

	return runStepGraph(ctx, "GuestJourneyPaymentFlow", steps.NewNoopStep(), waitForInvoiceStep, map[steps.Step][]domain.WeightedTuple[steps.Step]{
		waitForInvoiceStep:          {{Value: payInvoiceStep, Weight: 1}},
		payInvoiceStep:              {{Value: waitForConfirmedBookingStep, Weight: 1}},
		waitForConfirmedBookingStep: nil,
	}, timeBetweenStepsInSeconds)
}
