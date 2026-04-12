package services

import (
	"context"
	"testing"
	"time"
)

type fakeNotificationTimeEventService struct {
	registerCh   chan chan<- time.Time
	unregisterCh chan chan<- time.Time
	current      time.Time
	hasCurrent   bool
}

func (f *fakeNotificationTimeEventService) Start(context.Context) {}

func (f *fakeNotificationTimeEventService) Register(TimeEventType, chan<- time.Time) {
	panic("unreachable")
}

func (f *fakeNotificationTimeEventService) Unregister(TimeEventType, chan<- time.Time) {
	panic("unreachable")
}

func (f *fakeNotificationTimeEventService) CurrentTime() (time.Time, bool) {
	return f.current, f.hasCurrent
}

func (f *fakeNotificationTimeEventService) register(_ timeEventType, ch chan<- time.Time) {
	f.registerCh <- ch
}

func (f *fakeNotificationTimeEventService) unregister(_ timeEventType, ch chan<- time.Time) {
	f.unregisterCh <- ch
}

type testHourTimeEventService struct {
	*fakeNotificationTimeEventService
}

func (f testHourTimeEventService) Register(eventType TimeEventType, ch chan<- time.Time) {
	f.fakeNotificationTimeEventService.register(eventType, ch)
}

func (f testHourTimeEventService) Unregister(eventType TimeEventType, ch chan<- time.Time) {
	f.fakeNotificationTimeEventService.unregister(eventType, ch)
}

func TestHourNotificationServiceForwardsMatchingHoursAndUnregisters(t *testing.T) {
	fake := &fakeNotificationTimeEventService{
		registerCh:   make(chan chan<- time.Time, 1),
		unregisterCh: make(chan chan<- time.Time, 1),
	}
	service := NewHourNotificationService(testHourTimeEventService{fake})

	timerCh := make(chan interface{}, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		service.HourNotification(ctx, timerCh, 15)
		close(done)
	}()

	events := <-fake.registerCh
	events <- time.Date(2026, time.April, 7, 14, 0, 0, 0, time.UTC)

	select {
	case got := <-timerCh:
		t.Fatalf("unexpected notification: %#v", got)
	default:
	}

	expected := time.Date(2026, time.April, 7, 15, 0, 0, 0, time.UTC)
	events <- expected

	select {
	case raw := <-timerCh:
		got, ok := raw.(time.Time)
		if !ok {
			t.Fatalf("unexpected notification type: %T", raw)
		}
		if !got.Equal(expected) {
			t.Fatalf("unexpected notification time: %s", got)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for hour notification")
	}

	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for service to stop")
	}

	select {
	case unregistered := <-fake.unregisterCh:
		if unregistered != events {
			t.Fatal("service unregistered a different channel")
		}
	default:
		t.Fatal("expected unregister to be called")
	}
}

func TestHourNotificationServiceCurrentTimeDelegatesToTimeEventService(t *testing.T) {
	expected := time.Date(2026, time.April, 7, 9, 30, 0, 0, time.UTC)
	service := NewHourNotificationService(testHourTimeEventService{
		&fakeNotificationTimeEventService{current: expected, hasCurrent: true},
	})

	got, ok := service.CurrentTime()
	if !ok {
		t.Fatal("expected current time")
	}
	if !got.Equal(expected) {
		t.Fatalf("unexpected current time: %s", got)
	}
}
