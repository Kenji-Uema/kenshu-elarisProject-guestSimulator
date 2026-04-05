package journey

import (
	"context"
	"fmt"
	"strings"

	"github.com/Kenji-Uema/guestSimulator/internal/app/flows"
	"github.com/Kenji-Uema/guestSimulator/internal/app/journey/journey_services"
	"github.com/Kenji-Uema/guestSimulator/internal/domain"
	"github.com/Kenji-Uema/guestSimulator/internal/domain/dto/guest_registration"
	"github.com/Kenji-Uema/guestSimulator/internal/infra/telemetry"
	"github.com/Kenji-Uema/guestSimulator/internal/port"
	"github.com/brianvoe/gofakeit/v7"
)

type GuestJourney struct {
	state             *domain.State
	cache             port.Cache
	cacheService      *journey_services.JourneyCacheService
	guestRegisterFlow *flows.Flow
	bookingFlow       *flows.Flow
	paymentFlow       *flows.Flow
	lodgingFlow       *flows.LodgingFlow
	rabbitConsumer    port.RabbitConsumer
	communication     *journey_services.GuestCommunicationBus
}

func NewGuestJourney(state *domain.State,
	guestRegisterFlow *flows.Flow, bookingFlow *flows.Flow, paymentFlow *flows.Flow, lodgingFlow *flows.LodgingFlow,
	rabbitConsumer port.RabbitConsumer, cache port.Cache) (*GuestJourney, error) {

	communication := journey_services.NewGuestCommunicationBus()

	if state == nil || guestRegisterFlow == nil || bookingFlow == nil || paymentFlow == nil || lodgingFlow == nil || rabbitConsumer == nil || cache == nil {
		return nil, fmt.Errorf("invalid guest journey dependencies")
	}

	return &GuestJourney{
		state:             state,
		cache:             cache,
		cacheService:      journey_services.NewJourneyCacheService(cache),
		guestRegisterFlow: guestRegisterFlow,
		bookingFlow:       bookingFlow,
		paymentFlow:       paymentFlow,
		lodgingFlow:       lodgingFlow,
		rabbitConsumer:    rabbitConsumer,
		communication:     communication,
	}, nil
}

func (g *GuestJourney) Start(ctx context.Context) (err error) {
	ctx, span := telemetry.Tracer.Start(ctx, "GuestJourney")
	defer span.End()
	defer func() {
		if err == nil {
			return
		}

		if g.communication != nil && g.communication.Consumer != nil {
			_ = journey_services.CloseCommunication(ctx, g.state, g.communication)
			return
		}
		if g.rabbitConsumer != nil {
			_ = g.rabbitConsumer.CloseChannel()
		}
	}()

	if err := g.createGuest(ctx); err != nil {
		return err
	}
	if err := g.guestRegisterFlow.Start(ctx); err != nil {
		return err
	}
	if err := g.setupCommunication(ctx); err != nil {
		return err
	}
	if err := g.bookingFlow.Start(ctx); err != nil {
		return err
	}
	cacheValue, err := g.cache.Load(ctx, g.state)
	if err != nil {
		return err
	}
	if cacheValue.Booking == nil || cacheValue.Booking.BookingID == "" {
		return fmt.Errorf("booking flow finished without bookingId")
	}

	paymentWaitCtx, paymentWaitSpan := telemetry.Tracer.Start(ctx, "GuestJourneyPaymentWait")
	if err := g.cacheService.SyncStateCache(paymentWaitCtx, g.state, "UpdateBookingCacheStep"); err != nil {
		paymentWaitSpan.End()
		return err
	}
	if err := journey_services.WaitPaymentRequestMessage(paymentWaitCtx, g.state, g.cache, g.communication); err != nil {
		paymentWaitSpan.End()
		return err
	}
	paymentWaitSpan.End()

	paymentFlowCtx, paymentFlowSpan := telemetry.Tracer.Start(ctx, "GuestJourneyPaymentFlow")
	if err := g.paymentFlow.Start(paymentFlowCtx); err != nil {
		paymentFlowSpan.End()
		return err
	}
	paymentFlowSpan.End()

	bookingConfirmCtx, bookingConfirmSpan := telemetry.Tracer.Start(ctx, "GuestJourneyBookingConfirmationWait")
	if err := g.cacheService.SyncStateCache(bookingConfirmCtx, g.state, "UpdateInvoiceCacheStep"); err != nil {
		bookingConfirmSpan.End()
		return err
	}
	if err := journey_services.WaitBookingConfirmationMessage(bookingConfirmCtx, g.state, g.cache, g.communication); err != nil {
		bookingConfirmSpan.End()
		return err
	}
	bookingConfirmSpan.End()

	checkinWaitCtx, checkinWaitSpan := telemetry.Tracer.Start(ctx, "GuestJourneyLodgingPreparation")
	if err := journey_services.WaitCheckinTomorrowMessage(checkinWaitCtx, g.state, g.cache, g.communication); err != nil {
		checkinWaitSpan.End()
		return err
	}
	checkinWaitSpan.End()

	lodgingStayCtx, lodgingStaySpan := telemetry.Tracer.Start(ctx, "GuestJourneyLodgingStay")
	if err := flows.RunLodgingStayFlow(lodgingStayCtx, g.lodgingFlow); err != nil {
		lodgingStaySpan.End()
		return err
	}
	lodgingStaySpan.End()

	return g.cleanupGuestJourney(ctx)
}

func (g *GuestJourney) createGuest(ctx context.Context) error {
	ctx, span := telemetry.Tracer.Start(ctx, "SaveGuestCacheStep")
	defer span.End()

	name := strings.Split(gofakeit.Name(), " ")
	given := "Guest"
	surname := "Simulator"
	if len(name) > 0 && name[0] != "" {
		given = name[0]
	}
	if len(name) > 1 && name[1] != "" {
		surname = name[1]
	}

	guest := guest_registration.Guest{
		DocumentId:     gofakeit.SSN(),
		GivenNames:     given,
		Surname:        surname,
		Email:          fmt.Sprintf("%s.%s@test.com", given, surname),
		BillingAddress: gofakeit.Address().Address,
	}
	g.state.Guest = &guest
	g.state.RedisKey = fmt.Sprintf("guest.pending.%s", strings.ToLower(guest.DocumentId))

	return g.cacheService.InitializeStateCache(ctx, g.state)
}

func (g *GuestJourney) setupCommunication(ctx context.Context) error {
	ctx, span := telemetry.Tracer.Start(ctx, "GuestJourneyCommunication")
	defer span.End()

	return journey_services.SetupCommunication(ctx, g.state, g.rabbitConsumer, g.communication)
}

func (g *GuestJourney) cleanupGuestJourney(ctx context.Context) error {
	ctx, span := telemetry.Tracer.Start(ctx, "GuestJourneyCleanup")
	defer span.End()

	cacheLogCtx, cacheLogSpan := telemetry.Tracer.Start(ctx, "GuestJourneyCleanupInspectCache")
	defer cacheLogSpan.End()

	if err := g.cacheService.LogStateCache(cacheLogCtx, g.state); err != nil {
		return err
	}

	ctx, cleanupSpan := telemetry.Tracer.Start(ctx, "GuestJourneyCleanupDeleteRedis")
	defer cleanupSpan.End()

	cleanupSpan.SetAttributes(journey_services.CommunicationCleanupAttributes(g.state)...)
	return journey_services.CleanupState(ctx, g.state, g.cacheService, g.communication)
}
