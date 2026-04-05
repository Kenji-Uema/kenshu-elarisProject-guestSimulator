package services

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/Kenji-Uema/guestSimulator/internal/domain"
	timeevent "github.com/Kenji-Uema/guestSimulator/internal/domain/dto/time_event"
	amqp "github.com/rabbitmq/amqp091-go"
	"google.golang.org/protobuf/proto"
)

type timeEventType string

type TimeEventType = timeEventType

const TimeEventHourChange timeEventType = "hour_change"

type timeEventConsumer interface {
	Consume(ctx context.Context) (<-chan amqp.Delivery, error)
}

type TimeEventService interface {
	Start(ctx context.Context)
	Register(eventType TimeEventType, ch chan<- time.Time)
	Unregister(eventType TimeEventType, ch chan<- time.Time)
	CurrentTime() (time.Time, bool)
}

type timeEventService struct {
	hourChangeClient timeEventConsumer

	hourChangeChannelsMu sync.RWMutex
	hourChangeChannels   *domain.Set[chan<- time.Time]
	currentTimeMu        sync.RWMutex
	currentTime          time.Time
	hasCurrentTime       bool
}

func NewTimeEventService(hourChangeClient timeEventConsumer) (TimeEventService, error) {
	if hourChangeClient == nil {
		return nil, fmt.Errorf("hourChangeClient is nil")
	}

	return &timeEventService{
		hourChangeClient:   hourChangeClient,
		hourChangeChannels: domain.NewSet[chan<- time.Time](),
	}, nil
}

func NewInMemoryTimeEventService() TimeEventService {
	return &timeEventService{
		hourChangeChannels: domain.NewSet[chan<- time.Time](),
	}
}

func (s *timeEventService) Start(ctx context.Context) {
	hourChangeDeliveries, err := s.hourChangeClient.Consume(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "failed to consume hour change events", "error", err)
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		case hourChange, ok := <-hourChangeDeliveries:
			if !ok {
				slog.InfoContext(ctx, "hour change consumer closed")
				return
			}

			currentTime, err := s.unmarshalTimeEvent(ctx, hourChange.Body)
			if err != nil {
				slog.ErrorContext(ctx, "failed to unmarshal hour change event", "error", err)
				s.nackDelivery(ctx, hourChange, "hourChangeEvent")
				continue
			}

			s.ackDelivery(ctx, hourChange, "hourChangeEvent")
			s.notifyHourChange(ctx, currentTime)
		}
	}
}

func (s *timeEventService) Register(eventType timeEventType, ch chan<- time.Time) {
	if ch == nil {
		return
	}

	switch eventType {
	case TimeEventHourChange:
		s.hourChangeChannelsMu.Lock()
		defer s.hourChangeChannelsMu.Unlock()
		s.hourChangeChannels.Add(ch)
	}
}

func (s *timeEventService) Unregister(eventType timeEventType, ch chan<- time.Time) {
	if ch == nil {
		return
	}

	switch eventType {
	case TimeEventHourChange:
		s.hourChangeChannelsMu.Lock()
		defer s.hourChangeChannelsMu.Unlock()
		s.hourChangeChannels.Remove(ch)
	}
}

func (s *timeEventService) CurrentTime() (time.Time, bool) {
	s.currentTimeMu.RLock()
	defer s.currentTimeMu.RUnlock()

	if !s.hasCurrentTime {
		return time.Time{}, false
	}

	return s.currentTime, true
}

func (s *timeEventService) notifyHourChange(ctx context.Context, currentTime time.Time) {
	s.currentTimeMu.Lock()
	s.currentTime = currentTime
	s.hasCurrentTime = true
	s.currentTimeMu.Unlock()

	s.hourChangeChannelsMu.RLock()
	channels := s.hourChangeChannels.Values()
	s.hourChangeChannelsMu.RUnlock()

	s.notify(ctx, currentTime, channels, "hour change")
}

func (s *timeEventService) notify(ctx context.Context, currentTime time.Time, channels []chan<- time.Time, eventName string) {
	for _, ch := range channels {
		select {
		case ch <- currentTime:
			slog.DebugContext(ctx, "published "+eventName+" event to subscriber", "currentTime", currentTime)
		default:
			slog.WarnContext(ctx, "skipped "+eventName+" event for busy subscriber", "currentTime", currentTime)
		}
	}
}

func (s *timeEventService) ackDelivery(ctx context.Context, delivery amqp.Delivery, deliveryName string) {
	if err := delivery.Ack(false); err != nil {
		slog.ErrorContext(ctx, "failed to ack "+deliveryName, "error", err, "routingKey", delivery.RoutingKey)
	}
}

func (s *timeEventService) nackDelivery(ctx context.Context, delivery amqp.Delivery, deliveryName string) {
	if err := delivery.Nack(false, false); err != nil {
		slog.ErrorContext(ctx, "failed to nack "+deliveryName, "error", err, "routingKey", delivery.RoutingKey)
	}
}

func (s *timeEventService) unmarshalTimeEvent(ctx context.Context, body []byte) (time.Time, error) {
	var timeEvent timeevent.TimeEvent
	if err := proto.Unmarshal(body, &timeEvent); err != nil {
		slog.WarnContext(ctx, "invalid hour.changed payload", "error", err)
		return time.Time{}, err
	}

	if timeEvent.GetTime() == nil {
		return time.Time{}, fmt.Errorf("time event payload missing time")
	}

	return timeEvent.GetTime().AsTime(), nil
}
