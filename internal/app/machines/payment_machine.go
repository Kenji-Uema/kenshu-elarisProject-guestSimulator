package machines

import (
	"context"
	"fmt"

	"github.com/Kenji-Uema/guestSimulator/internal/app/steps"
	"github.com/Kenji-Uema/guestSimulator/internal/app/steps/payment_step"
	"github.com/Kenji-Uema/guestSimulator/internal/config"
	"github.com/Kenji-Uema/guestSimulator/internal/domain"
	"github.com/Kenji-Uema/guestSimulator/internal/port"
	transporthttp "github.com/Kenji-Uema/guestSimulator/internal/transport/http"
)

func NewPaymentMachineWithState(state *domain.State, machineConfig config.PaymentMachineConfig, serviceConfig config.ServicesConfig, cache port.Cache) (*Machine, error) {
	guestClient := transporthttp.NewRestyClient(serviceConfig.GuestManagerUrl, serviceConfig.GuestManagerPort)
	paymentClient := transporthttp.NewRestyClient(serviceConfig.PaymentSimulatorUrl, serviceConfig.PaymentSimulatorPort)

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

	return &Machine{
		spanName:                  "PaymentMachine",
		zeroStep:                  steps.NewInitStep(state),
		firstStep:                 flow.Start,
		stateMap:                  flow.StateMap(),
		timeBetweenStepsInSeconds: machineConfig.TimeBetweenStepsInSeconds,
	}, nil
}

func RunPaymentJourney(ctx context.Context, machine *Machine, timeBetweenStepsInSeconds int) error {
	if machine == nil {
		return fmt.Errorf("failed to compose payment journey from payment machine")
	}

	var waitForInvoiceStep steps.Step
	var payInvoiceStep steps.Step
	var waitForConfirmedBookingStep steps.Step

	if machine.firstStep != nil && machine.firstStep.Name() == "WaitForInvoiceStep" {
		waitForInvoiceStep = machine.firstStep
	}

	for step := range machine.stateMap {
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
		return fmt.Errorf("failed to compose payment journey from payment machine")
	}

	return runStepGraph(ctx, "GuestJourneyPaymentFlow", noopStep{}, waitForInvoiceStep, map[steps.Step][]domain.WeightedTuple[steps.Step]{
		waitForInvoiceStep:          {{Value: payInvoiceStep, Weight: 1}},
		payInvoiceStep:              {{Value: waitForConfirmedBookingStep, Weight: 1}},
		waitForConfirmedBookingStep: nil,
	}, timeBetweenStepsInSeconds)
}
