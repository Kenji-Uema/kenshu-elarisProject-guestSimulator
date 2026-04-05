package app

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Kenji-Uema/guestSimulator/internal/app/flows"
	"github.com/Kenji-Uema/guestSimulator/internal/app/steps"
	"github.com/Kenji-Uema/guestSimulator/internal/app/steps/journey_step"
	"github.com/Kenji-Uema/guestSimulator/internal/config"
	"github.com/Kenji-Uema/guestSimulator/internal/domain"
	"github.com/Kenji-Uema/guestSimulator/internal/infra/mq"
	"github.com/Kenji-Uema/guestSimulator/internal/infra/telemetry"
	"github.com/Kenji-Uema/guestSimulator/internal/port"
)

type GuestJourney struct {
	state                     *domain.State
	cache                     port.Cache
	guestRegisterFlow         *flows.Flow
	bookingFlow               *flows.Flow
	paymentFlow               *flows.Flow
	lodgingFlow               *flows.Flow
	steps                     journey_step.Steps
	timeBetweenStepsInSeconds int
}

func NewGuestJourney(flowConfig config.GuestJourneyFlowConfig, state *domain.State,
	guestRegisterFlow *flows.Flow, bookingFlow *flows.Flow, paymentFlow *flows.Flow, lodgingFlow *flows.Flow,
	rabbitConnection port.RabbitConnection, cache port.Cache) (*GuestJourney, error) {

	communication := &journey_step.GuestCommunicationRuntime{}

	if state == nil || guestRegisterFlow == nil || bookingFlow == nil || paymentFlow == nil || lodgingFlow == nil || rabbitConnection == nil || cache == nil {
		return nil, fmt.Errorf("invalid guest journey dependencies")
	}

	return &GuestJourney{
		state:                     state,
		cache:                     cache,
		guestRegisterFlow:         guestRegisterFlow,
		bookingFlow:               bookingFlow,
		paymentFlow:               paymentFlow,
		lodgingFlow:               lodgingFlow,
		steps:                     journey_step.NewSteps(state, cache, rabbitConnection, mq.ConsumerFactory{}, communication),
		timeBetweenStepsInSeconds: flowConfig.TimeBetweenStepsInSeconds,
	}, nil
}

func (g *GuestJourney) Run(ctx context.Context, concurrencyLevel int) {
	finishNotification := make(chan bool, concurrencyLevel)

	for i := 0; i < concurrencyLevel; i++ {
		g.startJourney(ctx, finishNotification)
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-finishNotification:
			g.startJourney(ctx, finishNotification)
		}
	}
}

func (g *GuestJourney) Start(ctx context.Context) error {
	ctx, span := telemetry.Tracer.Start(ctx, "GuestJourney")
	defer span.End()

	if err := g.guestRegisterFlow.Start(ctx); err != nil {
		return err
	}
	if err := g.setupCommunication(ctx); err != nil {
		return err
	}
	if err := g.runBooking(ctx); err != nil {
		return err
	}

	go g.finishJourney(ctx)
	return nil
}

func (g *GuestJourney) finishJourney(ctx context.Context) {
	if err := g.finishJourneySync(ctx); err != nil {
		slog.ErrorContext(ctx, "guest journey finish failed", "err", err)
	}
}

func (g *GuestJourney) finishJourneySync(ctx context.Context) error {
	if err := g.runPaymentFlow(ctx); err != nil {
		return err
	}
	if err := g.runLodgingFlow(ctx); err != nil {
		return err
	}
	return g.cleanupGuestJourney(ctx)
}

func (g *GuestJourney) setupCommunication(ctx context.Context) error {
	ctx, span := telemetry.Tracer.Start(ctx, "GuestJourneyCommunication")
	defer span.End()

	if err := g.executeJourneyStep(ctx, g.steps.Communication.SaveGuest); err != nil {
		return err
	}

	return g.executeJourneyStep(ctx, g.steps.Communication.SetupQueue)
}

func (g *GuestJourney) runBooking(ctx context.Context) error {
	ctx, span := telemetry.Tracer.Start(ctx, "GuestJourneyBooking")
	defer span.End()

	if err := flows.RunBookingFlow(ctx, g.bookingFlow, g.timeBetweenStepsInSeconds); err != nil {
		return err
	}
	cacheValue, err := g.cache.Load(ctx, g.state)
	if err != nil {
		return err
	}
	if cacheValue.Booking == nil || cacheValue.Booking.BookingID == "" {
		return fmt.Errorf("booking flow finished without bookingId")
	}

	return nil
}

func (g *GuestJourney) runPaymentFlow(ctx context.Context) error {
	ctx, span := telemetry.Tracer.Start(ctx, "GuestJourneyPayment")
	defer span.End()

	if err := g.waitForPaymentRequest(ctx); err != nil {
		return err
	}

	return g.runBookingPayment(ctx)
}

func (g *GuestJourney) waitForPaymentRequest(ctx context.Context) error {
	ctx, span := telemetry.Tracer.Start(ctx, "GuestJourneyPaymentWait")
	defer span.End()

	if err := g.executeJourneyStep(ctx, g.steps.PaymentWait.UpdateBookingCache); err != nil {
		return err
	}

	return g.executeJourneyStep(ctx, g.steps.PaymentWait.WaitPaymentRequest)
}

func (g *GuestJourney) runBookingPayment(ctx context.Context) error {
	ctx, span := telemetry.Tracer.Start(ctx, "GuestJourneyPaymentFlow")
	defer span.End()

	if err := flows.RunPaymentFlow(ctx, g.paymentFlow, g.timeBetweenStepsInSeconds); err != nil {
		return err
	}
	cacheValue, err := g.cache.Load(ctx, g.state)
	if err != nil {
		return err
	}
	if cacheValue.Invoice == nil || cacheValue.Invoice.InvoiceNumber == "" {
		return fmt.Errorf("payment flow finished without invoiceNumber")
	}

	return nil
}

func (g *GuestJourney) runLodgingFlow(ctx context.Context) error {
	ctx, span := telemetry.Tracer.Start(ctx, "GuestJourneyLodging")
	defer span.End()

	if err := g.waitForCheckInDay(ctx); err != nil {
		return err
	}

	return g.runLodgingStay(ctx)
}

func (g *GuestJourney) waitForCheckInDay(ctx context.Context) error {
	ctx, span := telemetry.Tracer.Start(ctx, "GuestJourneyLodgingPreparation")
	defer span.End()

	if err := g.executeJourneyStep(ctx, g.steps.StayWait.UpdateInvoiceCache); err != nil {
		return err
	}
	if err := g.executeJourneyStep(ctx, g.steps.StayWait.WaitBookingConfirmation); err != nil {
		return err
	}

	return g.executeJourneyStep(ctx, g.steps.StayWait.WaitCheckinTomorrow)
}

func (g *GuestJourney) runLodgingStay(ctx context.Context) error {
	ctx, span := telemetry.Tracer.Start(ctx, "GuestJourneyLodgingStay")
	defer span.End()

	return flows.RunLodgingStayFlow(ctx, g.lodgingFlow, g.timeBetweenStepsInSeconds)
}

func (g *GuestJourney) cleanupGuestJourney(ctx context.Context) error {
	ctx, span := telemetry.Tracer.Start(ctx, "GuestJourneyCleanup")
	defer span.End()

	if err := g.runCleanupStep(ctx, "GuestJourneyCleanupInspectCache", g.steps.Cleanup.LogCache); err != nil {
		return err
	}
	if err := g.runCleanupStep(ctx, "GuestJourneyCleanupDeleteRedis", g.steps.Cleanup.DeleteCache); err != nil {
		return err
	}

	return g.runCleanupStep(ctx, "GuestJourneyCleanupDeleteCommunicationQueue", g.steps.Cleanup.CloseQueue)
}

func (g *GuestJourney) executeJourneyStep(ctx context.Context, step steps.Step) error {
	if step == nil {
		return fmt.Errorf("journey step is nil")
	}
	if err := step.Validate(); err != nil {
		return err
	}
	return step.Execute(ctx)
}

func (g *GuestJourney) runCleanupStep(ctx context.Context, spanName string, step steps.Step) error {
	ctx, span := telemetry.Tracer.Start(ctx, spanName)
	defer span.End()

	return g.executeJourneyStep(ctx, step)
}

func (g *GuestJourney) startJourney(ctx context.Context, finishNotification chan<- bool) {
	go func() {
		if err := g.Start(ctx); err != nil {
			slog.ErrorContext(ctx, "guest journey stopped with error", "err", err)
		}
		finishNotification <- true
	}()
}
