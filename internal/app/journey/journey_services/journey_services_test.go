package journey_services

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/Kenji-Uema/guestSimulator/internal/config"
	"github.com/Kenji-Uema/guestSimulator/internal/domain"
	"github.com/Kenji-Uema/guestSimulator/internal/domain/dto"
	"github.com/Kenji-Uema/guestSimulator/internal/domain/dto/booking"
	"github.com/Kenji-Uema/guestSimulator/internal/domain/dto/communication"
	"github.com/Kenji-Uema/guestSimulator/internal/domain/dto/guest_registration"
	mqfakes "github.com/Kenji-Uema/guestSimulator/internal/infra/mq/fakes"
	redisfakes "github.com/Kenji-Uema/guestSimulator/internal/infra/redis/fakes"
	"github.com/Kenji-Uema/guestSimulator/internal/port"
	amqp "github.com/rabbitmq/amqp091-go"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestJourneyCacheServiceInitializeStateCache(t *testing.T) {
	cache := &redisfakes.Cache{}
	service := NewJourneyCacheService(cache)
	state := &domain.State{
		Guest: &guest_registration.Guest{
			Email: "guest@test.com",
		},
		RedisKey: "guest.pending.1",
	}

	if err := service.InitializeStateCache(context.Background(), state); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cache.SavedValue.PersonalInfo == nil || cache.SavedValue.PersonalInfo.Email != "guest@test.com" {
		t.Fatalf("unexpected saved cache value: %#v", cache.SavedValue)
	}
}

func TestJourneyCacheServiceSyncStateCache(t *testing.T) {
	cache := &redisfakes.Cache{
		LoadValue: dto.GuestJourneyCacheValue{
			Booking: &dto.GuestJourneyBooking{BookingID: "booking-1"},
		},
	}
	service := NewJourneyCacheService(cache)
	state := &domain.State{
		GuestId:  "guest-1",
		RedisKey: "guest.pending.1",
		Guest:    &guest_registration.Guest{Email: "guest@test.com"},
	}

	if err := service.SyncStateCache(context.Background(), state, "step"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cache.SavedValue.GuestID != "guest-1" {
		t.Fatalf("unexpected guest id: %q", cache.SavedValue.GuestID)
	}
	if cache.SavedValue.PersonalInfo == nil || cache.SavedValue.PersonalInfo.Email != "guest@test.com" {
		t.Fatalf("unexpected personal info: %#v", cache.SavedValue.PersonalInfo)
	}
}

func TestJourneyCacheServiceDeleteAndLogStateCache(t *testing.T) {
	cache := &redisfakes.Cache{GetValue: `{"guestId":"guest-1"}`}
	service := NewJourneyCacheService(cache)
	state := &domain.State{RedisKey: "guest.pending.1"}

	if err := service.LogStateCache(context.Background(), state); err != nil {
		t.Fatalf("unexpected log error: %v", err)
	}
	if err := service.DeleteStateCache(context.Background(), state); err != nil {
		t.Fatalf("unexpected delete error: %v", err)
	}
	if cache.DeletedKey != "guest.pending.1" {
		t.Fatalf("unexpected deleted key: %q", cache.DeletedKey)
	}
}

func TestDeliveryHeaderHandlesDifferentValueTypes(t *testing.T) {
	delivery := amqp.Delivery{
		Headers: amqp.Table{
			"string": "value",
			"bytes":  []byte("bytes"),
			"int":    123,
		},
	}

	if got := deliveryHeader(delivery, "string"); got != "value" {
		t.Fatalf("unexpected string header: %q", got)
	}
	if got := deliveryHeader(delivery, "bytes"); got != "bytes" {
		t.Fatalf("unexpected bytes header: %q", got)
	}
	if got := deliveryHeader(delivery, "int"); got != "123" {
		t.Fatalf("unexpected int header: %q", got)
	}
	if got := deliveryHeader(delivery, "missing"); got != "" {
		t.Fatalf("unexpected missing header: %q", got)
	}
}

func TestSetupCommunicationConfiguresConsumerAndStartsBus(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	state := &domain.State{GuestId: "guest-1"}
	consumer := &mqfakes.Consumer{Deliveries: make(chan amqp.Delivery, 1)}
	bus := NewGuestCommunicationBus()

	err := SetupCommunication(ctx, state, consumer, config.RabbitMqConsumerConfig{}, bus)
	if err != nil {
		t.Fatalf("unexpected setup error: %v", err)
	}

	if state.QueueName != guestQueuePrefix+"guest-1" {
		t.Fatalf("unexpected queue name: %q", state.QueueName)
	}
	if state.RoutingKey != guestRoutingKeyPrefix+"guest-1" {
		t.Fatalf("unexpected routing key: %q", state.RoutingKey)
	}
	if consumer.DeclareCfg.Name != state.QueueName || consumer.DeclareCfg.Durable || !consumer.DeclareCfg.AutoDelete {
		t.Fatalf("unexpected declare config: %#v", consumer.DeclareCfg)
	}
	if consumer.BindingCfg.ExchangeName != guestCommunicationExchange || consumer.BindingCfg.RoutingKey != state.RoutingKey {
		t.Fatalf("unexpected binding config: %#v", consumer.BindingCfg)
	}
	if bus.Consumer == nil {
		t.Fatal("expected bus consumer to be set")
	}

	msg := &communication.PaymentRequest{
		InvoiceNumber: "invoice-1",
		Total:         &communication.Money{Amount: 100, Currency: "USD"},
		Booking:       &communication.BookingSummary{CottageName: "Alps"},
		Payer:         &communication.PayerSummary{Email: "guest@test.com"},
	}
	body, err := proto.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal message: %v", err)
	}
	acker := &mqfakes.Acknowledger{}
	consumer.Deliveries <- amqp.Delivery{
		Acknowledger: acker,
		Headers:      amqp.Table{"message_type": paymentRequestMessageType},
		Body:         body,
	}

	var pending *communication.PaymentRequest
	waitFor(t, func() bool {
		msg, ok := takePendingPaymentRequest(bus)
		if ok {
			pending = msg
			return true
		}
		return false
	})

	if pending == nil || pending.GetInvoiceNumber() != "invoice-1" {
		t.Fatalf("unexpected pending payment request: %#v", pending)
	}
	if acker.AckCalls != 1 || acker.NackCalls != 0 {
		t.Fatalf("unexpected ack state: ack=%d nack=%d", acker.AckCalls, acker.NackCalls)
	}
}

func TestGuestCommunicationBusHandleDeliveryRejectsUnsupportedType(t *testing.T) {
	err := NewGuestCommunicationBus().handleDelivery(context.Background(), amqp.Delivery{
		Headers: amqp.Table{"message_type": "unknown"},
	})
	if err == nil || !strings.Contains(err.Error(), `unsupported message_type "unknown"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCloseCommunicationResetsBus(t *testing.T) {
	consumer := &mqfakes.Consumer{}
	bus := NewGuestCommunicationBus()
	bus.Consumer = consumer
	bus.pendingPaymentRequests = []*communication.PaymentRequest{{InvoiceNumber: "invoice-1"}}
	state := &domain.State{QueueName: "queue", RoutingKey: "routing"}

	if err := CloseCommunication(context.Background(), state, bus); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if consumer.CloseCalls != 1 {
		t.Fatalf("unexpected close calls: %d", consumer.CloseCalls)
	}
	if bus.Consumer != nil {
		t.Fatal("expected bus consumer to be cleared")
	}
	if len(bus.pendingPaymentRequests) != 0 {
		t.Fatalf("expected payment requests to be reset: %#v", bus.pendingPaymentRequests)
	}
}

func TestCleanupStateDeletesCacheAndClosesCommunication(t *testing.T) {
	cache := &redisfakes.Cache{}
	service := NewJourneyCacheService(cache)
	consumer := &mqfakes.Consumer{}
	bus := NewGuestCommunicationBus()
	bus.Consumer = consumer
	state := &domain.State{
		RedisKey:   "guest.pending.1",
		QueueName:  "queue",
		RoutingKey: "routing",
	}

	if err := CleanupState(context.Background(), state, service, bus); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cache.DeletedKey != "guest.pending.1" {
		t.Fatalf("unexpected deleted key: %q", cache.DeletedKey)
	}
	if consumer.CloseCalls != 1 {
		t.Fatalf("unexpected close calls: %d", consumer.CloseCalls)
	}
}

func TestWaitPaymentRequestMessageUsesPendingQueue(t *testing.T) {
	state := &domain.State{GuestId: "guest-1"}
	cache := &redisfakes.Cache{
		LoadValue: dto.GuestJourneyCacheValue{
			PersonalInfo: &guest_registration.Guest{Email: "guest@test.com"},
			Booking:      &dto.GuestJourneyBooking{SelectedCottage: "Alps"},
		},
	}
	bus := NewGuestCommunicationBus()
	bus.pendingPaymentRequests = []*communication.PaymentRequest{{
		InvoiceNumber: "invoice-1",
		Total:         &communication.Money{Amount: 100, Currency: "USD"},
		Booking:       &communication.BookingSummary{CottageName: "Alps"},
		Payer:         &communication.PayerSummary{Email: "guest@test.com"},
	}}

	if err := WaitPaymentRequestMessage(context.Background(), state, cache, bus); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cache.SavedValue.Invoice == nil || cache.SavedValue.Invoice.InvoiceNumber != "invoice-1" {
		t.Fatalf("unexpected invoice cache: %#v", cache.SavedValue.Invoice)
	}
}

func TestWaitBookingConfirmationMessageUsesSubscriberChannel(t *testing.T) {
	state := &domain.State{GuestId: "guest-1"}
	cache := &redisfakes.Cache{
		LoadValue: dto.GuestJourneyCacheValue{
			Booking: &dto.GuestJourneyBooking{BookingID: "booking-1"},
		},
	}
	bus := NewGuestCommunicationBus()
	msg := &communication.BookingConfirmedNotificationEvent{
		BookingId:     "booking-1",
		BookingStatus: communication.BookingStatus_BOOKING_STATUS_CONFIRMED,
		Guest:         &communication.Guest{GuestId: "guest-1"},
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	go func() {
		waitForBusSubscriber(func() int {
			bus.bookingConfirmationMu.RLock()
			defer bus.bookingConfirmationMu.RUnlock()
			return len(bus.bookingConfirmationChannels.Values())
		})
		bus.publishBookingConfirmation(context.Background(), msg)
	}()

	if err := WaitBookingConfirmationMessage(ctx, state, cache, bus); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWaitCheckinTodayMessageUsesPendingQueue(t *testing.T) {
	checkIn := time.Date(2026, time.April, 12, 0, 0, 0, 0, time.UTC)
	state := &domain.State{GuestId: "guest-1"}
	cache := &redisfakes.Cache{
		LoadValue: dto.GuestJourneyCacheValue{
			Booking: &dto.GuestJourneyBooking{
				BookingID:       "booking-1",
				SelectedCottage: "Alps",
				SelectedPeriod:  &booking.Period{Start: checkIn, End: checkIn.AddDate(0, 0, 3)},
			},
		},
	}
	bus := NewGuestCommunicationBus()
	bus.pendingCheckinToday = []*communication.CheckInTodayNotification{{
		BookingId:   "booking-1",
		GuestId:     "guest-1",
		CottageName: "Alps",
		CheckIn:     timestamppb.New(checkIn),
	}}

	if err := WaitCheckinTodayMessage(context.Background(), state, cache, bus); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMatchPaymentRequestRejectsMismatchedEmail(t *testing.T) {
	cache := &redisfakes.Cache{
		LoadValue: dto.GuestJourneyCacheValue{
			PersonalInfo: &guest_registration.Guest{Email: "expected@test.com"},
			Booking:      &dto.GuestJourneyBooking{SelectedCottage: "Alps"},
		},
	}

	err := matchPaymentRequest(context.Background(), &domain.State{}, cache, &communication.PaymentRequest{
		InvoiceNumber: "invoice-1",
		Total:         &communication.Money{Amount: 100, Currency: "USD"},
		Booking:       &communication.BookingSummary{CottageName: "Alps"},
		Payer:         &communication.PayerSummary{Email: "other@test.com"},
	})
	if err == nil || !strings.Contains(err.Error(), "payer does not match guest email") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCommunicationCleanupAttributes(t *testing.T) {
	attrs := CommunicationCleanupAttributes(&domain.State{
		QueueName:  "queue",
		RoutingKey: "routing",
	})
	if len(attrs) != 3 {
		t.Fatalf("unexpected attribute count: %d", len(attrs))
	}
}

func waitFor(t *testing.T, condition func() bool) {
	t.Helper()

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatal("timed out waiting for condition")
}

func waitForBusSubscriber(length func() int) {
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if length() > 0 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
}

var (
	_ port.Cache          = (*redisfakes.Cache)(nil)
	_ port.RabbitConsumer = (*mqfakes.Consumer)(nil)
	_ amqp.Acknowledger   = (*mqfakes.Acknowledger)(nil)
)
