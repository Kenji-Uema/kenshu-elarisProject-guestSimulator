package config

import "github.com/Kenji-Uema/guestSimulator/internal/domain"

type LodgingActionGate struct {
	HasNotBeforeHour    bool
	NotBeforeHour       int
	SystemNotification  domain.SystemNotification
	SystemRequest       domain.SystemRequest
	WaitForNotification domain.SystemNotification
}

type LodgingPlannedAction struct {
	Action domain.GuestAction
	Gate   LodgingActionGate
}

type LodgingResponseStep struct {
	Request domain.SystemRequest
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
				{Action: domain.GuestActionShowForCheckin, Gate: LodgingActionGate{HasNotBeforeHour: true, NotBeforeHour: 15}},
			},
			ShowDocument:      LodgingResponseStep{Request: domain.SystemRequestDocument},
			ShowBookingNumber: LodgingResponseStep{Request: domain.SystemRequestBookingNumber},
		},
		FirstDayStay: []LodgingPlannedAction{
			{Action: domain.GuestActionEnterCottage},
			{Action: domain.GuestActionGoForABath, Gate: LodgingActionGate{HasNotBeforeHour: true, NotBeforeHour: 16}},
			{Action: domain.GuestActionGoForDinner, Gate: LodgingActionGate{HasNotBeforeHour: true, NotBeforeHour: 18, SystemNotification: domain.SystemNotificationDinnerReady}},
			{Action: domain.GuestActionGoToSleep, Gate: LodgingActionGate{HasNotBeforeHour: true, NotBeforeHour: 22}},
		},
		RecurringStay: []LodgingPlannedAction{
			{Action: domain.GuestActionWakeup, Gate: LodgingActionGate{HasNotBeforeHour: true, NotBeforeHour: 7}},
			{Action: domain.GuestActionGoForBreakfast, Gate: LodgingActionGate{HasNotBeforeHour: true, NotBeforeHour: 8, SystemNotification: domain.SystemNotificationBreakfastReady}},
			{Action: domain.GuestActionLeaveCleanupNotice, Gate: LodgingActionGate{HasNotBeforeHour: true, NotBeforeHour: 10}},
			{Action: domain.GuestActionEnjoyResort, Gate: LodgingActionGate{HasNotBeforeHour: true, NotBeforeHour: 11}},
			{Action: domain.GuestActionGoForABath, Gate: LodgingActionGate{HasNotBeforeHour: true, NotBeforeHour: 16}},
			{Action: domain.GuestActionGoForDinner, Gate: LodgingActionGate{HasNotBeforeHour: true, NotBeforeHour: 18, SystemNotification: domain.SystemNotificationDinnerReady}},
			{Action: domain.GuestActionGoToSleep, Gate: LodgingActionGate{HasNotBeforeHour: true, NotBeforeHour: 22}},
		},
		Checkout: []LodgingPlannedAction{
			{Action: domain.GuestActionWakeup, Gate: LodgingActionGate{HasNotBeforeHour: true, NotBeforeHour: 7}},
			{Action: domain.GuestActionLeaveCottage, Gate: LodgingActionGate{HasNotBeforeHour: true, NotBeforeHour: 9}},
			{Action: domain.GuestActionProceedToCheckout},
			{Action: domain.GuestActionReturnCottageKey, Gate: LodgingActionGate{SystemRequest: domain.SystemRequestCottageKey, WaitForNotification: domain.SystemNotificationCheckOutComplete}},
		},
	}
}
