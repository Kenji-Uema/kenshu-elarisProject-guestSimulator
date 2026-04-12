package config

import (
	"context"
	"reflect"
	"testing"

	"github.com/Kenji-Uema/guestSimulator/internal/app/steps"
	"github.com/Kenji-Uema/guestSimulator/internal/domain/dto/lodging"
)

type testStep struct {
	name string
}

func (s testStep) Validate() error               { return nil }
func (s testStep) Execute(context.Context) error { return nil }
func (s testStep) Name() string                  { return s.name }

func TestSecretString(t *testing.T) {
	if got := Secret("super-secret").String(); got != "REDACTED" {
		t.Fatalf("unexpected secret string: %q", got)
	}
}

func TestDefaultBookingFlow(t *testing.T) {
	flowSteps := BookingSteps{
		ListCottages:  testStep{name: "list"},
		SelectCottage: testStep{name: "select-cottage"},
		SelectPeriod:  testStep{name: "select-period"},
		BookCottage:   testStep{name: "book"},
		End:           testStep{name: "end"},
	}

	flow := DefaultBookingFlow(flowSteps)

	if flow.Start != flowSteps.ListCottages {
		t.Fatalf("unexpected start step: %#v", flow.Start)
	}

	stateMap := flow.StateMap()
	if len(stateMap) != 4 {
		t.Fatalf("unexpected state map size: %d", len(stateMap))
	}

	got := stateMap[flowSteps.SelectPeriod]
	want := flow.SelectPeriodTransitions
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected select period transitions: %#v", got)
	}
}

func TestDefaultGuestRegisterFlow(t *testing.T) {
	flowSteps := GuestRegisterFlowSteps{
		RegisterGuest: testStep{name: "register"},
		RetrieveGuest: testStep{name: "retrieve"},
		End:           testStep{name: "end"},
	}

	flow := DefaultGuestRegisterFlow(flowSteps)

	if flow.Start != flowSteps.RegisterGuest {
		t.Fatalf("unexpected start step: %#v", flow.Start)
	}

	if got := flow.StateMap()[flowSteps.RetrieveGuest]; len(got) != 1 || got[0].Value != flowSteps.End || got[0].Weight != 1.0 {
		t.Fatalf("unexpected retrieve transitions: %#v", got)
	}
}

func TestDefaultPaymentFlow(t *testing.T) {
	flowSteps := PaymentSteps{
		PayInvoice: testStep{name: "pay"},
		End:        testStep{name: "end"},
	}

	flow := DefaultPaymentFlow(flowSteps)

	if flow.Start != flowSteps.PayInvoice {
		t.Fatalf("unexpected start step: %#v", flow.Start)
	}

	if got := flow.StateMap()[flowSteps.PayInvoice]; len(got) != 1 || got[0].Value != flowSteps.End || got[0].Weight != 1.0 {
		t.Fatalf("unexpected pay transitions: %#v", got)
	}
}

func TestDefaultLodgingFlow(t *testing.T) {
	flow := DefaultLodgingFlow()

	if len(flow.Checkin.ShowUp) == 0 {
		t.Fatal("expected checkin show-up actions")
	}
	if flow.Checkin.ShowUp[0].Action != lodging.GuestAction_SHOW_FOR_CHECKIN {
		t.Fatalf("unexpected first checkin action: %v", flow.Checkin.ShowUp[0].Action)
	}
	if !flow.Checkin.ShowUp[0].Gate.HasNotBeforeHour || flow.Checkin.ShowUp[0].Gate.NotBeforeHour != 15 {
		t.Fatalf("unexpected first checkin gate: %#v", flow.Checkin.ShowUp[0].Gate)
	}
	if len(flow.Checkout) == 0 {
		t.Fatal("expected checkout actions")
	}

	last := flow.Checkout[len(flow.Checkout)-1]
	if last.Action != lodging.GuestAction_RETURN_COTTAGE_KEY {
		t.Fatalf("unexpected last checkout action: %v", last.Action)
	}
	if last.Gate.SystemRequest != lodging.SystemRequest_REQUEST_COTTAGE_KEY {
		t.Fatalf("unexpected checkout request gate: %v", last.Gate.SystemRequest)
	}
	if last.Gate.WaitForNotification != lodging.SystemNotification_CHECK_OUT_COMPLETE {
		t.Fatalf("unexpected checkout notification gate: %v", last.Gate.WaitForNotification)
	}
}

var _ steps.Step = testStep{}
