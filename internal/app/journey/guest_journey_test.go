package journey

import (
	"context"
	"strings"
	"testing"

	"github.com/Kenji-Uema/guestSimulator/internal/app/flows"
	"github.com/Kenji-Uema/guestSimulator/internal/app/journey/journey_services"
	"github.com/Kenji-Uema/guestSimulator/internal/config"
	"github.com/Kenji-Uema/guestSimulator/internal/domain"
	mqfakes "github.com/Kenji-Uema/guestSimulator/internal/infra/mq/fakes"
	redisfakes "github.com/Kenji-Uema/guestSimulator/internal/infra/redis/fakes"
	"github.com/Kenji-Uema/guestSimulator/internal/port"
	amqp "github.com/rabbitmq/amqp091-go"
)

func TestNewGuestJourneyRejectsMissingDependencies(t *testing.T) {
	journey, err := NewGuestJourney(nil, &flows.Flow{}, &flows.Flow{}, &flows.Flow{}, &flows.LodgingFlow{}, &mqfakes.Consumer{}, &redisfakes.Cache{}, config.RabbitMqConsumerConfig{})
	if err == nil {
		t.Fatal("expected constructor error")
	}
	if journey != nil {
		t.Fatalf("expected nil journey, got %#v", journey)
	}
}

func TestCreateGuestInitializesStateAndCache(t *testing.T) {
	cache := &redisfakes.Cache{}
	state := &domain.State{}
	journey := &GuestJourney{
		state:        state,
		cache:        cache,
		cacheService: journey_services.NewJourneyCacheService(cache),
	}

	if err := journey.createGuest(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if state.Guest == nil {
		t.Fatal("expected guest to be created")
	}
	if state.Guest.Email == "" || state.Guest.DocumentId == "" {
		t.Fatalf("unexpected guest data: %#v", state.Guest)
	}
	if !strings.HasPrefix(state.RedisKey, "guest.pending.") {
		t.Fatalf("unexpected redis key: %q", state.RedisKey)
	}
	if cache.SavedValue.PersonalInfo == nil || cache.SavedValue.PersonalInfo.Email != state.Guest.Email {
		t.Fatalf("unexpected cached personal info: %#v", cache.SavedValue.PersonalInfo)
	}
}

func TestGuestJourneySetupCommunicationConfiguresGuestQueue(t *testing.T) {
	state := &domain.State{GuestId: "guest-1"}
	consumer := &mqfakes.Consumer{Deliveries: make(chan amqp.Delivery)}
	journey := &GuestJourney{
		state:            state,
		rabbitConsumer:   consumer,
		communicationCfg: config.RabbitMqConsumerConfig{},
		communication:    journey_services.NewGuestCommunicationBus(),
	}

	if err := journey.setupCommunication(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if state.QueueName == "" || state.RoutingKey == "" {
		t.Fatalf("expected communication fields to be set: %#v", state)
	}
	if consumer.DeclareCfg.Name != state.QueueName {
		t.Fatalf("unexpected declare config: %#v", consumer.DeclareCfg)
	}
}

func TestGuestJourneyCleanupGuestJourneyDeletesStateAndClosesCommunication(t *testing.T) {
	cache := &redisfakes.Cache{GetValue: `{"guestId":"guest-1"}`}
	consumer := &mqfakes.Consumer{}
	journey := &GuestJourney{
		state: &domain.State{
			RedisKey:   "guest.pending.1",
			QueueName:  "queue",
			RoutingKey: "routing",
		},
		cacheService: journey_services.NewJourneyCacheService(cache),
		communication: &journey_services.GuestCommunicationBus{
			Consumer: consumer,
		},
	}

	if err := journey.cleanupGuestJourney(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cache.DeletedKey != "guest.pending.1" {
		t.Fatalf("unexpected deleted key: %q", cache.DeletedKey)
	}
	if consumer.CloseCalls != 1 {
		t.Fatalf("unexpected close calls: %d", consumer.CloseCalls)
	}
}

func TestGuestJourneyStartReturnsRegisterFlowErrorAfterCreatingGuest(t *testing.T) {
	state := &domain.State{}
	cache := &redisfakes.Cache{GetValue: `{"guestId":"guest-1"}`}
	serviceCfg := config.ServicesConfig{
		GuestManagerUrl:      "127.0.0.1",
		GuestManagerPort:     1,
		CottageManagerUrl:    "127.0.0.1",
		CottageManagerPort:   1,
		PaymentSimulatorUrl:  "127.0.0.1",
		PaymentSimulatorPort: 1,
	}

	guestRegisterFlow, err := flows.NewGuestRegisterFlowWithState(state, serviceCfg, cache)
	if err != nil {
		t.Fatalf("unexpected register flow error: %v", err)
	}
	bookingFlow, err := flows.NewBookingFlow(state, serviceCfg, cache, nil)
	if err != nil {
		t.Fatalf("unexpected booking flow error: %v", err)
	}
	paymentFlow, err := flows.NewPaymentFlowWithState(state, serviceCfg, cache)
	if err != nil {
		t.Fatalf("unexpected payment flow error: %v", err)
	}
	lodgingFlow, err := flows.NewLodgingFlowWithState(state, serviceCfg, cache, nil)
	if err != nil {
		t.Fatalf("unexpected lodging flow error: %v", err)
	}

	journey, err := NewGuestJourney(state, guestRegisterFlow, bookingFlow, paymentFlow, lodgingFlow, &mqfakes.Consumer{}, cache, config.RabbitMqConsumerConfig{})
	if err != nil {
		t.Fatalf("unexpected guest journey error: %v", err)
	}

	err = journey.Start(context.Background())
	if err == nil {
		t.Fatal("expected start error")
	}
	if state.Guest == nil || state.RedisKey == "" {
		t.Fatalf("expected guest state to be initialized before failure: %#v", state)
	}
	if cache.SavedValue.PersonalInfo == nil {
		t.Fatalf("expected guest to be cached: %#v", cache.SavedValue)
	}
}

var (
	_ port.Cache          = (*redisfakes.Cache)(nil)
	_ port.RabbitConsumer = (*mqfakes.Consumer)(nil)
)
