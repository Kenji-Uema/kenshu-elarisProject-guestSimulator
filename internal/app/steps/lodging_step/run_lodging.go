package lodging_step

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/Kenji-Uema/guestSimulator/internal/app/steps"
	"github.com/Kenji-Uema/guestSimulator/internal/config"
	"github.com/Kenji-Uema/guestSimulator/internal/domain"
	"github.com/Kenji-Uema/guestSimulator/internal/domain/dto"
	"github.com/Kenji-Uema/guestSimulator/internal/infra/telemetry"
	"github.com/Kenji-Uema/guestSimulator/internal/port"
	ws "github.com/Kenji-Uema/guestSimulator/internal/transport/websocket"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type RunLodgingStep struct {
	state               *domain.State
	chatURL             string
	cache               port.Cache
	notificationService hourNotificationService
	flow                config.LodgingFlow
}

type actionPlanStep struct {
	dayOffset int
	action    domain.GuestAction
	gate      config.LodgingActionGate
}

type hourNotificationService interface {
	HourNotification(ctx context.Context, timerCh chan interface{}, hour int)
	CurrentTime() (time.Time, bool)
}

func NewRunLodgingStep(state *domain.State, chatURL string, cache port.Cache, notificationService hourNotificationService, flow config.LodgingFlow) steps.Step {
	return &RunLodgingStep{
		state:               state,
		chatURL:             chatURL,
		cache:               cache,
		notificationService: notificationService,
		flow:                flow,
	}
}

func (s RunLodgingStep) Name() string {
	return "RunLodgingStep"
}

func (s RunLodgingStep) Validate() error {
	if s.state == nil {
		return fmt.Errorf("invalid state, state is nil")
	}
	if s.cache == nil {
		return fmt.Errorf("invalid guest journey cache")
	}
	if s.notificationService == nil {
		return fmt.Errorf("invalid hour notification service")
	}
	if s.chatURL == "" {
		return fmt.Errorf("invalid lodging chat url")
	}
	if len(s.flow.Checkin.ShowUp) == 0 {
		return fmt.Errorf("invalid lodging flow, checkin show up plan is empty")
	}

	return nil
}

func (s RunLodgingStep) Execute(ctx context.Context) error {
	ctx, span := telemetry.Tracer.Start(ctx, "RunLodgingStep")
	defer span.End()

	cacheValue, err := s.loadCache(ctx)
	if err != nil {
		return err
	}
	span.SetAttributes(
		attribute.String("guest.id", s.state.GuestId),
		attribute.String("booking.id", cacheValue.Booking.BookingID),
		attribute.String("booking.cottage", cacheValue.Booking.SelectedCottage),
	)

	client, err := ws.NewClient(ctx, s.chatURL)
	if err != nil {
		return err
	}
	defer func() {
		if err := client.Close(); err != nil {
			slog.WarnContext(ctx, "failed to close lodging websocket", "err", err)
		}
	}()

	keyID := uuid.NewString()

	if err := s.executeActionPlan(ctx, "checkin_show_up", client, s.expandPlan(0, s.flow.Checkin.ShowUp), cacheValue); err != nil {
		return err
	}

	if err := s.finishCheckin(ctx, client, cacheValue); err != nil {
		return err
	}

	msg, err := s.expectRequest(ctx, client, domain.SystemRequestGiveCottageKey)
	if err != nil {
		return err
	}
	if err := client.Reply(ctx, msg, &domain.GuestResponse{
		ReceiveCottageKey: &domain.ReceiveCottageKey{CottageKeyID: keyID},
	}); err != nil {
		return err
	}
	if err := client.SendAction(ctx, domain.GuestActionTakeCottageKey); err != nil {
		return err
	}
	if err := s.executeActionPlan(ctx, "stay", client, s.buildStayActionPlan(cacheValue), cacheValue); err != nil {
		return err
	}
	if err := s.executeCheckoutPlan(ctx, client, keyID, cacheValue); err != nil {
		return err
	}

	return nil
}

func (s RunLodgingStep) loadCache(ctx context.Context) (dto.GuestJourneyCacheValue, error) {
	cacheValue, err := s.cache.Load(ctx, s.state)
	if err != nil {
		return dto.GuestJourneyCacheValue{}, err
	}
	if cacheValue.PersonalInfo == nil {
		return dto.GuestJourneyCacheValue{}, fmt.Errorf("invalid cached guest context")
	}
	if cacheValue.Booking == nil || cacheValue.Booking.BookingID == "" || cacheValue.Booking.SelectedPeriod == nil {
		return dto.GuestJourneyCacheValue{}, fmt.Errorf("invalid cached booking context")
	}

	return cacheValue, nil
}

func (s RunLodgingStep) buildStayActionPlan(cacheValue dto.GuestJourneyCacheValue) []actionPlanStep {
	plan := s.expandPlan(0, s.flow.FirstDayStay)

	for offset := 2; offset <= s.fullStayDays(cacheValue); offset++ {
		plan = append(plan, s.expandPlan(offset, s.flow.RecurringStay)...)
	}

	return plan
}

func (s RunLodgingStep) buildCheckoutActionPlan(cacheValue dto.GuestJourneyCacheValue) []actionPlanStep {
	return s.expandPlan(s.checkoutDayOffset(cacheValue), s.flow.Checkout)
}

func (s RunLodgingStep) expandPlan(dayOffset int, actions []config.LodgingPlannedAction) []actionPlanStep {
	plan := make([]actionPlanStep, 0, len(actions))
	for _, action := range actions {
		plan = append(plan, actionPlanStep{
			dayOffset: dayOffset,
			action:    action.Action,
			gate:      action.Gate,
		})
	}

	return plan
}

func (s RunLodgingStep) executeActionPlan(ctx context.Context, phase string, client *ws.Client, plan []actionPlanStep, cacheValue dto.GuestJourneyCacheValue) error {
	slog.InfoContext(ctx, "executing action plan", "phase", phase, "plan_length", len(plan))
	for _, step := range plan {
		if _, err := s.waitForActionGate(ctx, client, step, cacheValue); err != nil {
			return err
		}

		actionCtx, actionSpan := telemetry.Tracer.Start(ctx, "SendLodgingAction")
		actionSpan.SetAttributes(
			attribute.String("lodging.action", string(step.action)),
			attribute.Int("lodging.day_offset", step.dayOffset),
		)
		slog.InfoContext(ctx, "sending planned lodging action",
			"action", step.action,
			"day_offset", step.dayOffset)

		if err := client.SendAction(actionCtx, step.action); err != nil {
			actionSpan.End()
			return err
		}
		if step.gate.WaitForNotification != "" {
			if err := s.expectNotification(actionCtx, client, step.gate.WaitForNotification); err != nil {
				actionSpan.End()
				return err
			}
		}
		actionSpan.End()
	}

	return nil
}

func (s RunLodgingStep) executeCheckoutPlan(ctx context.Context, client *ws.Client, keyID string, cacheValue dto.GuestJourneyCacheValue) error {
	for _, step := range s.buildCheckoutActionPlan(cacheValue) {
		actionCtx, actionSpan := telemetry.Tracer.Start(ctx, "ExecuteCheckoutAction")
		actionSpan.SetAttributes(
			attribute.String("lodging.action", string(step.action)),
			attribute.Int("lodging.day_offset", step.dayOffset),
		)

		msg, err := s.waitForActionGate(actionCtx, client, step, cacheValue)
		if err != nil {
			actionSpan.End()
			return err
		}

		if step.gate.SystemRequest == domain.SystemRequestCottageKey {
			if err := client.Reply(actionCtx, msg, &domain.GuestResponse{
				ReturnCottageKey: &domain.ReturnCottageKey{CottageKeyID: keyID},
			}); err != nil {
				actionSpan.End()
				return err
			}
		}

		slog.InfoContext(actionCtx, "sending planned checkout action",
			"action", step.action,
			"day_offset", step.dayOffset)

		if err := client.SendAction(actionCtx, step.action); err != nil {
			actionSpan.End()
			return err
		}
		if step.gate.WaitForNotification != "" {
			if err := s.expectNotification(actionCtx, client, step.gate.WaitForNotification); err != nil {
				actionSpan.End()
				return err
			}
		}
		actionSpan.End()
	}

	return nil
}

func (s RunLodgingStep) waitForActionGate(ctx context.Context, client *ws.Client, step actionPlanStep, cacheValue dto.GuestJourneyCacheValue) (*domain.ChatMessage, error) {
	ctx, span := telemetry.Tracer.Start(ctx, "WaitForActionGate")
	defer span.End()

	day := startOfUTCDay(cacheValue.Booking.SelectedPeriod.Start).AddDate(0, 0, step.dayOffset)
	span.SetAttributes(
		attribute.String("lodging.action", string(step.action)),
		attribute.Int("lodging.day_offset", step.dayOffset),
		attribute.String("lodging.gate.day", day.Format(time.DateOnly)),
	)

	if step.gate.HasNotBeforeHour {
		target := time.Date(day.UTC().Year(), day.UTC().Month(), day.UTC().Day(), step.gate.NotBeforeHour, 0, 0, 0, time.UTC)
		span.SetAttributes(attribute.Int("lodging.gate.not_before_hour", step.gate.NotBeforeHour))
		if err := s.waitUntil(ctx, target); err != nil {
			return nil, err
		}
	}

	if step.gate.SystemNotification != "" {
		span.SetAttributes(attribute.String("lodging.gate.notification", string(step.gate.SystemNotification)))
		if err := s.expectNotification(ctx, client, step.gate.SystemNotification); err != nil {
			return nil, err
		}
	}

	if step.gate.SystemRequest != "" {
		span.SetAttributes(attribute.String("lodging.gate.request", string(step.gate.SystemRequest)))
		msg, err := s.expectRequest(ctx, client, step.gate.SystemRequest)
		if err != nil {
			return nil, err
		}
		return msg, nil
	}

	return nil, nil
}

func (s RunLodgingStep) waitUntil(ctx context.Context, target time.Time) error {
	ctx, span := telemetry.Tracer.Start(ctx, "WaitUntil")
	defer span.End()

	span.SetAttributes(
		attribute.String("wait.target", target.UTC().Format(time.RFC3339)),
		attribute.Int("wait.target_hour", target.UTC().Hour()),
	)

	notifications := make(chan interface{}, 1)
	go s.notificationService.HourNotification(ctx, notifications, target.Hour())

	if now, ok := s.notificationService.CurrentTime(); ok {
		span.SetAttributes(attribute.String("wait.current", now.UTC().Format(time.RFC3339)))
		if !now.UTC().Before(target) {
			span.AddEvent("wait already satisfied")
			return nil
		}
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case rawEvent, ok := <-notifications:
			if !ok {
				return context.Canceled
			}
			eventTime, ok := rawEvent.(time.Time)
			if !ok {
				return fmt.Errorf("unexpected hour notification type %T", rawEvent)
			}
			span.AddEvent("hour notification received", trace.WithAttributes(attribute.String("event.time", eventTime.UTC().Format(time.RFC3339))))
			if sameUTCDay(eventTime, target) && !eventTime.UTC().Before(target) {
				span.AddEvent("wait target reached")
				return nil
			}
		}
	}
}

func (s RunLodgingStep) fullStayDays(cacheValue dto.GuestJourneyCacheValue) int {
	checkIn := startOfUTCDay(cacheValue.Booking.SelectedPeriod.Start)
	checkOut := startOfUTCDay(cacheValue.Booking.SelectedPeriod.End)

	nights := int(checkOut.Sub(checkIn) / (24 * time.Hour))
	if nights <= 1 {
		return 0
	}

	return nights - 1
}

func (s RunLodgingStep) checkoutDayOffset(cacheValue dto.GuestJourneyCacheValue) int {
	checkIn := startOfUTCDay(cacheValue.Booking.SelectedPeriod.Start)
	checkOut := startOfUTCDay(cacheValue.Booking.SelectedPeriod.End)

	nights := int(checkOut.Sub(checkIn) / (24 * time.Hour))
	if nights < 0 {
		return 0
	}

	return nights
}

func startOfUTCDay(t time.Time) time.Time {
	t = t.UTC()
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func sameUTCDay(left time.Time, right time.Time) bool {
	left = left.UTC()
	right = right.UTC()

	return left.Year() == right.Year() &&
		left.Month() == right.Month() &&
		left.Day() == right.Day()
}

func (s RunLodgingStep) finishCheckin(ctx context.Context, client *ws.Client, cacheValue dto.GuestJourneyCacheValue) error {
	ctx, span := telemetry.Tracer.Start(ctx, "FinishCheckin")
	defer span.End()

	span.SetAttributes(attribute.String("booking.id", cacheValue.Booking.BookingID))
	for {
		msg, err := client.WaitForNextSystemMessage(ctx)
		if err != nil {
			return err
		}

		switch {
		case msg.SystemNotification == domain.SystemNotificationBookingChecking:
			span.AddEvent("booking checking notification received")
			continue
		case msg.SystemNotification == domain.SystemNotificationCheckInComplete:
			span.AddEvent("check-in completed")
			return nil
		case s.flow.Checkin.ShowDocument.Request != "" && msg.SystemRequest == s.flow.Checkin.ShowDocument.Request:
			span.AddEvent("show document request received")
			if err := client.Reply(ctx, msg, &domain.GuestResponse{
				ShowDocument: &domain.ShowDocument{DocumentID: cacheValue.PersonalInfo.DocumentId},
			}); err != nil {
				return err
			}
		case s.flow.Checkin.ShowBookingNumber.Request != "" && msg.SystemRequest == s.flow.Checkin.ShowBookingNumber.Request:
			span.AddEvent("show booking number request received")
			if err := client.Reply(ctx, msg, &domain.GuestResponse{
				ShowBookingNumber: &domain.ShowBookingNumber{BookingID: cacheValue.Booking.BookingID},
			}); err != nil {
				return err
			}
		default:
			slog.DebugContext(ctx, "ignoring websocket message during checkin", "message", msg)
		}
	}
}

func (s RunLodgingStep) expectNotification(ctx context.Context, client *ws.Client, notification domain.SystemNotification) error {
	ctx, span := telemetry.Tracer.Start(ctx, "WaitSystemNotification")
	defer span.End()

	span.SetAttributes(attribute.String("message.notification.expected", string(notification)))
	for {
		msg, err := client.WaitForNextSystemMessage(ctx)
		if err != nil {
			return err
		}

		if msg.SystemNotification == notification {
			span.AddEvent("expected system notification received")
			return nil
		}

		slog.DebugContext(ctx, "ignoring unrelated system notification", "expected", notification, "received", msg.SystemNotification)
	}
}

func (s RunLodgingStep) expectRequest(ctx context.Context, client *ws.Client, request domain.SystemRequest) (*domain.ChatMessage, error) {
	ctx, span := telemetry.Tracer.Start(ctx, "WaitSystemRequest")
	defer span.End()

	span.SetAttributes(attribute.String("message.request.expected", string(request)))
	for {
		msg, err := client.WaitForNextSystemMessage(ctx)
		if err != nil {
			return nil, err
		}

		if msg.SystemRequest == request {
			span.AddEvent("expected system request received")
			return msg, nil
		}

		slog.DebugContext(ctx, "ignoring unrelated system request", "expected", request, "received", msg.SystemRequest)
	}
}
