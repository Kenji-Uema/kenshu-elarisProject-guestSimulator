package machines

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Kenji-Uema/guestSimulator/internal/app/steps"
	"github.com/Kenji-Uema/guestSimulator/internal/app/steps/booking_step"
	"github.com/Kenji-Uema/guestSimulator/internal/app/steps/register_guest_step"
	"github.com/Kenji-Uema/guestSimulator/internal/config"
	"github.com/Kenji-Uema/guestSimulator/internal/domain"
	"github.com/Kenji-Uema/guestSimulator/internal/infra/clock"
	"github.com/Kenji-Uema/guestSimulator/internal/infra/http"
	"github.com/Kenji-Uema/guestSimulator/internal/port"
	"github.com/go-resty/resty/v2"
)

func NewBookingMachineWithState(state *domain.State, machineConfig config.BookingMachineConfig, serviceConfig config.ServicesConfig, cache port.Cache) (*Machine, error) {
	cottageClient := http.NewRestyClient(serviceConfig.CottageManagerUrl, serviceConfig.CottageManagerPort)
	guestClient := http.NewRestyClient(serviceConfig.GuestManagerUrl, serviceConfig.GuestManagerPort)
	clockEmu, err := clock.NewClockEmu(serviceConfig)
	if err != nil {
		return nil, err
	}
	return buildBookingMachine(state, cottageClient, guestClient, cache, clockEmu, machineConfig.TimeBetweenStepsInSeconds, config.DefaultBookingFlow).machine, nil
}

type bookingFlowBuilder func(config.BookingSteps) config.BookingFlow

type bookingMachineParts struct {
	flow    config.BookingFlow
	machine *Machine
}

func RunBookingJourney(ctx context.Context, machine *Machine, timeBetweenStepsInSeconds int) error {
	if machine == nil {
		return fmt.Errorf("failed to compose booking journey from booking machine")
	}

	var listCottagesStep steps.Step
	var selectCottageStep steps.Step
	var selectPeriodStep steps.Step
	var registerGuestStep steps.Step
	var bookCottageStep steps.Step

	if machine.firstStep != nil && machine.firstStep.Name() == "ListCottagesStep" {
		listCottagesStep = machine.firstStep
	}

	for step := range machine.stateMap {
		if step == nil {
			continue
		}
		switch step.Name() {
		case "ListCottagesStep":
			listCottagesStep = step
		case "SelectCottageStep":
			selectCottageStep = step
		case "SelectPeriodStep":
			selectPeriodStep = step
		case "RegisterGuestStep":
			registerGuestStep = step
		case "BookCottageStep":
			bookCottageStep = step
		}
	}

	if listCottagesStep == nil || selectCottageStep == nil || selectPeriodStep == nil || registerGuestStep == nil || bookCottageStep == nil {
		return fmt.Errorf("failed to compose booking journey from booking machine")
	}

	executeStep := func(step steps.Step) error {
		if step == nil {
			return fmt.Errorf("booking journey step is nil")
		}
		if err := step.Validate(); err != nil {
			return err
		}
		return step.Execute(ctx)
	}

	ticker := time.NewTicker(time.Duration(timeBetweenStepsInSeconds) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := executeStep(listCottagesStep); err != nil {
				return err
			}
			if err := executeStep(selectCottageStep); err != nil {
				return err
			}
			if err := executeStep(selectPeriodStep); err != nil {
				if errors.Is(err, booking_step.ErrNoSuitablePeriod) {
					continue
				}
				return err
			}
			if err := executeStep(bookCottageStep); err != nil {
				return err
			}
			return nil
		}
	}
}

func buildBookingMachine(state *domain.State, cottageClient *resty.Client, guestClient *resty.Client, cache port.Cache, clockEmu port.Clock,
	timeBetweenStepsInSeconds int, flowBuilder bookingFlowBuilder) bookingMachineParts {
	bookingMachineStates := map[string]steps.Step{
		"End":           steps.NewEndStep(state),
		"SelectCottage": booking_step.NewSelectCottageStep(state, cache),
		"ListCottages":  booking_step.NewListCottagesStep(state, cottageClient),
		"SelectPeriod":  booking_step.NewSelectPeriodStep(state, clockEmu, cottageClient, cache),
		"RegisterGuest": register_guest_step.NewRegisterGuestStep(guestClient, cache, state),
		"BookCottage":   booking_step.NewBookCottageStep(state, cottageClient, cache),
	}

	flow := flowBuilder(config.BookingSteps{
		End:           bookingMachineStates["End"],
		ListCottages:  bookingMachineStates["ListCottages"],
		SelectCottage: bookingMachineStates["SelectCottage"],
		SelectPeriod:  bookingMachineStates["SelectPeriod"],
		RegisterGuest: bookingMachineStates["RegisterGuest"],
		BookCottage:   bookingMachineStates["BookCottage"],
	})

	return bookingMachineParts{
		flow: flow,
		machine: &Machine{
			spanName:                  "BookingMachine",
			zeroStep:                  steps.NewInitStep(state),
			firstStep:                 flow.Start,
			stateMap:                  flow.StateMap(),
			timeBetweenStepsInSeconds: timeBetweenStepsInSeconds,
		},
	}
}
