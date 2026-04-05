package journey_step

import (
	"github.com/Kenji-Uema/guestSimulator/internal/app/steps"
	"github.com/Kenji-Uema/guestSimulator/internal/domain"
	"github.com/Kenji-Uema/guestSimulator/internal/port"
)

type CommunicationSteps struct {
	SaveGuest  steps.Step
	SetupQueue steps.Step
}

type PaymentWaitSteps struct {
	UpdateBookingCache steps.Step
	WaitPaymentRequest steps.Step
}

type StayWaitSteps struct {
	UpdateInvoiceCache      steps.Step
	WaitBookingConfirmation steps.Step
	WaitCheckinTomorrow     steps.Step
}

type CleanupSteps struct {
	LogCache    steps.Step
	DeleteCache steps.Step
	CloseQueue  steps.Step
}

type Steps struct {
	Communication CommunicationSteps
	PaymentWait   PaymentWaitSteps
	StayWait      StayWaitSteps
	Cleanup       CleanupSteps
}

func NewSteps(state *domain.State, cache port.Cache, rabbitConnection port.RabbitConnection, rabbitConsumerFactory port.RabbitConsumerFactory, communication *GuestCommunicationRuntime) Steps {
	return Steps{
		Communication: CommunicationSteps{
			SaveGuest:  NewSaveGuestCacheStep(state, cache),
			SetupQueue: NewSetupGuestCommunicationStep(state, rabbitConnection, rabbitConsumerFactory, communication),
		},
		PaymentWait: PaymentWaitSteps{
			UpdateBookingCache: NewUpdateBookingCacheStep(state, cache),
			WaitPaymentRequest: NewWaitPaymentRequestStep(state, cache, communication),
		},
		StayWait: StayWaitSteps{
			UpdateInvoiceCache:      NewUpdateInvoiceCacheStep(state, cache),
			WaitBookingConfirmation: NewWaitBookingConfirmationStep(state, cache, communication),
			WaitCheckinTomorrow:     NewWaitCheckinTomorrowStep(state, cache, communication),
		},
		Cleanup: CleanupSteps{
			LogCache:    NewLogGuestCacheStep(state, cache),
			DeleteCache: NewDeleteGuestCacheStep(state, cache),
			CloseQueue:  NewCloseGuestCommunicationStep(state, communication),
		},
	}
}
