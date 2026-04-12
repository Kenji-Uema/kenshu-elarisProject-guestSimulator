package services

import (
	"context"
	"testing"
	"time"

	"github.com/Kenji-Uema/guestSimulator/internal/domain"
	"github.com/Kenji-Uema/guestSimulator/internal/domain/dto/time_event"
	mqfakes "github.com/Kenji-Uema/guestSimulator/internal/infra/mq/fakes"
	amqp "github.com/rabbitmq/amqp091-go"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type fakeTimeEventConsumer struct {
	deliveries chan amqp.Delivery
	err        error
}

func (f fakeTimeEventConsumer) Consume(context.Context) (<-chan amqp.Delivery, error) {
	return f.deliveries, f.err
}

func TestNewTimeEventServiceRejectsNilConsumer(t *testing.T) {
	service, err := NewTimeEventService(nil)
	if err == nil {
		t.Fatal("expected error for nil consumer")
	}
	if service != nil {
		t.Fatalf("expected nil service, got %#v", service)
	}
}

func TestTimeEventServiceNotifyHourChangeUpdatesCurrentTimeAndSubscribers(t *testing.T) {
	service := &timeEventService{
		hourChangeChannels: domain.NewSet[chan<- time.Time](),
	}
	events := make(chan time.Time, 1)
	service.Register(TimeEventHourChange, events)

	now := time.Date(2026, time.April, 7, 14, 0, 0, 0, time.UTC)
	service.notifyHourChange(context.Background(), now)

	gotCurrent, ok := service.CurrentTime()
	if !ok {
		t.Fatal("expected current time to be available")
	}
	if !gotCurrent.Equal(now) {
		t.Fatalf("unexpected current time: %s", gotCurrent)
	}

	select {
	case got := <-events:
		if !got.Equal(now) {
			t.Fatalf("unexpected notification time: %s", got)
		}
	default:
		t.Fatal("expected hour change notification")
	}
}

func TestTimeEventServiceUnregisterRemovesSubscriber(t *testing.T) {
	service := &timeEventService{
		hourChangeChannels: domain.NewSet[chan<- time.Time](),
	}
	events := make(chan time.Time, 1)
	service.Register(TimeEventHourChange, events)
	service.Unregister(TimeEventHourChange, events)

	service.notifyHourChange(context.Background(), time.Date(2026, time.April, 7, 14, 0, 0, 0, time.UTC))

	select {
	case <-events:
		t.Fatal("did not expect notification after unregister")
	default:
	}
}

func TestTimeEventServiceStartAcknowledgesValidMessages(t *testing.T) {
	deliveries := make(chan amqp.Delivery, 1)
	serviceIface, err := NewTimeEventService(fakeTimeEventConsumer{deliveries: deliveries})
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}
	service := serviceIface.(*timeEventService)
	events := make(chan time.Time, 1)
	service.Register(TimeEventHourChange, events)

	done := make(chan struct{})
	go func() {
		service.Start(context.Background())
		close(done)
	}()

	expected := time.Date(2026, time.April, 7, 16, 0, 0, 0, time.UTC)
	body, err := proto.Marshal(&time_event.TimeEvent{Time: timestamppb.New(expected)})
	if err != nil {
		t.Fatalf("marshal time event: %v", err)
	}

	acker := &mqfakes.Acknowledger{}
	deliveries <- amqp.Delivery{
		Acknowledger: acker,
		DeliveryTag:  1,
		Body:         body,
	}
	close(deliveries)

	select {
	case got := <-events:
		if !got.Equal(expected) {
			t.Fatalf("unexpected event time: %s", got)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for notification")
	}

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for service to stop")
	}

	if acker.AckCalls != 1 {
		t.Fatalf("unexpected ack count: %d", acker.AckCalls)
	}
	if acker.NackCalls != 0 {
		t.Fatalf("unexpected nack count: %d", acker.NackCalls)
	}
}

func TestTimeEventServiceStartNacksInvalidMessages(t *testing.T) {
	deliveries := make(chan amqp.Delivery, 1)
	serviceIface, err := NewTimeEventService(fakeTimeEventConsumer{deliveries: deliveries})
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}

	done := make(chan struct{})
	go func() {
		serviceIface.Start(context.Background())
		close(done)
	}()

	acker := &mqfakes.Acknowledger{}
	deliveries <- amqp.Delivery{
		Acknowledger: acker,
		DeliveryTag:  2,
		Body:         []byte("not-protobuf"),
	}
	close(deliveries)

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for service to stop")
	}

	if acker.AckCalls != 0 {
		t.Fatalf("unexpected ack count: %d", acker.AckCalls)
	}
	if acker.NackCalls != 1 {
		t.Fatalf("unexpected nack count: %d", acker.NackCalls)
	}
}
