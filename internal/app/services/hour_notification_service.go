package services

import (
	"context"
	"log/slog"
	"time"
)

type HourNotificationService interface {
	HourNotification(ctx context.Context, timerCh chan interface{}, hour int)
	CurrentTime() (time.Time, bool)
}

type hourNotificationService struct {
	timeEventService TimeEventService
}

func NewHourNotificationService(timeEventService TimeEventService) HourNotificationService {
	return &hourNotificationService{timeEventService: timeEventService}
}

func (n hourNotificationService) HourNotification(ctx context.Context, timerCh chan interface{}, hour int) {
	events := make(chan time.Time, 1)
	n.timeEventService.Register(TimeEventHourChange, events)
	defer n.timeEventService.Unregister(TimeEventHourChange, events)

	for {
		select {
		case <-ctx.Done():
			return
		case eventTime, ok := <-events:
			if !ok {
				return
			}
			if eventTime.Hour() == hour {
				select {
				case timerCh <- eventTime:
				case <-ctx.Done():
					slog.DebugContext(ctx, "canceled while publishing hour notification", "error", ctx.Err())
					return
				}
			}
		}
	}
}

func (n hourNotificationService) CurrentTime() (time.Time, bool) {
	return n.timeEventService.CurrentTime()
}
