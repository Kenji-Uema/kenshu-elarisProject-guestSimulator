package journey_step

import (
	"github.com/Kenji-Uema/guestSimulator/internal/app/steps"
	"github.com/Kenji-Uema/guestSimulator/internal/domain"
	"github.com/Kenji-Uema/guestSimulator/internal/infra/mq"
	redisc "github.com/Kenji-Uema/guestSimulator/internal/infra/redis"
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

func NewSteps(state *domain.State, redisClient *redisc.Redis, rabbitConnection *mq.RabbitMqConnection, communication *GuestCommunicationRuntime) Steps {
	return Steps{
		Communication: CommunicationSteps{
			SaveGuest:  NewSaveGuestCacheStep(state, redisClient),
			SetupQueue: NewSetupGuestCommunicationStep(state, rabbitConnection, communication),
		},
		PaymentWait: PaymentWaitSteps{
			UpdateBookingCache: NewUpdateBookingCacheStep(state, redisClient),
			WaitPaymentRequest: NewWaitPaymentRequestStep(state, redisClient, communication),
		},
		StayWait: StayWaitSteps{
			UpdateInvoiceCache:      NewUpdateInvoiceCacheStep(state, redisClient),
			WaitBookingConfirmation: NewWaitBookingConfirmationStep(state, redisClient, communication),
			WaitCheckinTomorrow:     NewWaitCheckinTomorrowStep(state, redisClient, communication),
		},
		Cleanup: CleanupSteps{
			LogCache:    NewLogGuestCacheStep(state, redisClient),
			DeleteCache: NewDeleteGuestCacheStep(state, redisClient),
			CloseQueue:  NewCloseGuestCommunicationStep(state, communication),
		},
	}
}
