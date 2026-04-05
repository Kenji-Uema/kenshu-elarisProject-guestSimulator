package config

import "github.com/Kenji-Uema/guestSimulator/internal/domain/dto/lodging"

type LodgingActionGate struct {
	HasNotBeforeHour    bool
	NotBeforeHour       int
	SystemNotification  lodging.SystemNotification
	SystemRequest       lodging.SystemRequest
	WaitForNotification lodging.SystemNotification
}

type LodgingPlannedAction struct {
	Action lodging.GuestAction
	Gate   LodgingActionGate
}

type LodgingResponseStep struct {
	Request lodging.SystemRequest
}

type LodgingCheckinFlow struct {
	ShowUp            []LodgingPlannedAction
	ShowDocument      LodgingResponseStep
	ShowBookingNumber LodgingResponseStep
}

type LodgingFlow struct {
	Checkin       LodgingCheckinFlow
	FirstDayStay  []LodgingPlannedAction
	RecurringStay []LodgingPlannedAction
	Checkout      []LodgingPlannedAction
}

func DefaultLodgingFlow() LodgingFlow {
	return LodgingFlow{
		Checkin: LodgingCheckinFlow{
			ShowUp: []LodgingPlannedAction{
				{Action: lodging.GuestAction_SHOW_FOR_CHECKIN, Gate: LodgingActionGate{HasNotBeforeHour: true, NotBeforeHour: 15}},
			},
			ShowDocument:      LodgingResponseStep{Request: lodging.SystemRequest_REQUEST_DOCUMENT},
			ShowBookingNumber: LodgingResponseStep{Request: lodging.SystemRequest_REQUEST_BOOKING_NUMBER},
		},
		FirstDayStay: []LodgingPlannedAction{
			{Action: lodging.GuestAction_ENTER_COTTAGE},
			{Action: lodging.GuestAction_GO_FOR_A_BATH, Gate: LodgingActionGate{HasNotBeforeHour: true, NotBeforeHour: 16}},
			{Action: lodging.GuestAction_GO_FOR_DINNER, Gate: LodgingActionGate{HasNotBeforeHour: true, NotBeforeHour: 18, SystemNotification: lodging.SystemNotification_DINNER_READY}},
			{Action: lodging.GuestAction_GO_TO_SLEEP, Gate: LodgingActionGate{HasNotBeforeHour: true, NotBeforeHour: 22}},
		},
		RecurringStay: []LodgingPlannedAction{
			{Action: lodging.GuestAction_WAKEUP, Gate: LodgingActionGate{HasNotBeforeHour: true, NotBeforeHour: 7}},
			{Action: lodging.GuestAction_GO_FOR_BREAKFAST, Gate: LodgingActionGate{HasNotBeforeHour: true, NotBeforeHour: 8, SystemNotification: lodging.SystemNotification_BREAKFAST_READY}},
			{Action: lodging.GuestAction_LEAVE_CLEANUP_NOTIFICATION, Gate: LodgingActionGate{HasNotBeforeHour: true, NotBeforeHour: 10}},
			{Action: lodging.GuestAction_ENJOY_RESORT, Gate: LodgingActionGate{HasNotBeforeHour: true, NotBeforeHour: 11}},
			{Action: lodging.GuestAction_GO_FOR_A_BATH, Gate: LodgingActionGate{HasNotBeforeHour: true, NotBeforeHour: 16}},
			{Action: lodging.GuestAction_GO_FOR_DINNER, Gate: LodgingActionGate{HasNotBeforeHour: true, NotBeforeHour: 18, SystemNotification: lodging.SystemNotification_DINNER_READY}},
			{Action: lodging.GuestAction_GO_TO_SLEEP, Gate: LodgingActionGate{HasNotBeforeHour: true, NotBeforeHour: 22}},
		},
		Checkout: []LodgingPlannedAction{
			{Action: lodging.GuestAction_WAKEUP, Gate: LodgingActionGate{HasNotBeforeHour: true, NotBeforeHour: 7}},
			{Action: lodging.GuestAction_LEAVE_COTTAGE, Gate: LodgingActionGate{HasNotBeforeHour: true, NotBeforeHour: 9}},
			{Action: lodging.GuestAction_PROCEED_TO_CHECKOUT},
			{Action: lodging.GuestAction_RETURN_COTTAGE_KEY, Gate: LodgingActionGate{SystemRequest: lodging.SystemRequest_REQUEST_COTTAGE_KEY, WaitForNotification: lodging.SystemNotification_CHECK_OUT_COMPLETE}},
		},
	}
}
