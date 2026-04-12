package lodging_step

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Kenji-Uema/guestSimulator/internal/app/services"
	"github.com/Kenji-Uema/guestSimulator/internal/config"
	"github.com/Kenji-Uema/guestSimulator/internal/domain"
	"github.com/Kenji-Uema/guestSimulator/internal/domain/dto"
	bookingdto "github.com/Kenji-Uema/guestSimulator/internal/domain/dto/booking"
	"github.com/Kenji-Uema/guestSimulator/internal/domain/dto/guest_registration"
	"github.com/Kenji-Uema/guestSimulator/internal/domain/dto/lodging"
	redisfakes "github.com/Kenji-Uema/guestSimulator/internal/infra/redis/fakes"
	websocketfakes "github.com/Kenji-Uema/guestSimulator/internal/infra/websocket/fakes"
	"github.com/Kenji-Uema/guestSimulator/internal/port"
)

type fakeHourNotificationService struct {
	current    time.Time
	hasCurrent bool
	events     []interface{}
	hours      []int
}

func (s *fakeHourNotificationService) HourNotification(ctx context.Context, timerCh chan interface{}, hour int) {
	s.hours = append(s.hours, hour)
	for _, event := range s.events {
		select {
		case <-ctx.Done():
			return
		case timerCh <- event:
		}
	}
}

func (s *fakeHourNotificationService) CurrentTime() (time.Time, bool) {
	return s.current, s.hasCurrent
}

type fakeChatClientFactory struct {
	client port.LodgingChatClient
	err    error
	url    string
}

func (f *fakeChatClientFactory) NewClient(_ context.Context, url string) (port.LodgingChatClient, error) {
	f.url = url
	return f.client, f.err
}

func TestRunLodgingStepValidateRejectsMissingDependencies(t *testing.T) {
	err := RunLodgingStep{}.Validate()
	if err == nil || !strings.Contains(err.Error(), "state is nil") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNewRunLodgingStepReturnsNamedStep(t *testing.T) {
	step := NewRunLodgingStep(&domain.State{}, "ws://guest", &redisfakes.Cache{}, &fakeChatClientFactory{}, &fakeHourNotificationService{}, config.DefaultLodgingFlow())
	if step == nil {
		t.Fatal("expected step")
	}
	if step.Name() != "RunLodgingStep" {
		t.Fatalf("unexpected step name: %q", step.Name())
	}
}

func TestRunLodgingStepTimeHelpers(t *testing.T) {
	checkIn := time.Date(2026, time.April, 12, 18, 0, 0, 0, time.FixedZone("offset", -3*60*60))
	checkOut := checkIn.AddDate(0, 0, 3)
	cacheValue := dto.GuestJourneyCacheValue{
		Booking: &dto.GuestJourneyBooking{
			SelectedPeriod: &bookingdto.Period{Start: checkIn, End: checkOut},
		},
	}
	step := RunLodgingStep{}

	if step.fullStayDays(cacheValue) != 2 {
		t.Fatalf("unexpected full stay days: %d", step.fullStayDays(cacheValue))
	}
	if step.checkoutDayOffset(cacheValue) != 3 {
		t.Fatalf("unexpected checkout day offset: %d", step.checkoutDayOffset(cacheValue))
	}
	if !sameUTCDay(checkIn, checkIn.UTC()) {
		t.Fatal("expected same UTC day")
	}
	if start := startOfUTCDay(checkIn); start.Hour() != 0 || start.Location() != time.UTC {
		t.Fatalf("unexpected UTC day start: %s", start)
	}
}

func TestRunLodgingStepLoadCacheRejectsInvalidContexts(t *testing.T) {
	step := RunLodgingStep{
		cache: &redisfakes.Cache{LoadValue: dto.GuestJourneyCacheValue{}},
		state: &domain.State{},
	}

	if _, err := step.loadCache(context.Background()); err == nil || !strings.Contains(err.Error(), "invalid cached guest context") {
		t.Fatalf("unexpected error: %v", err)
	}

	step.cache = &redisfakes.Cache{LoadValue: dto.GuestJourneyCacheValue{
		PersonalInfo: &guest_registration.Guest{DocumentId: "doc-1"},
	}}
	if _, err := step.loadCache(context.Background()); err == nil || !strings.Contains(err.Error(), "invalid cached booking context") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunLodgingStepWaitUntilReturnsImmediatelyWhenAlreadySatisfied(t *testing.T) {
	target := time.Date(2026, time.April, 12, 15, 0, 0, 0, time.UTC)
	service := &fakeHourNotificationService{
		current:    target,
		hasCurrent: true,
	}
	step := RunLodgingStep{notificationService: service}

	if err := step.waitUntil(context.Background(), target); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunLodgingStepWaitUntilConsumesNotifications(t *testing.T) {
	target := time.Date(2026, time.April, 12, 15, 0, 0, 0, time.UTC)
	service := &fakeHourNotificationService{
		current:    target.Add(-time.Hour),
		hasCurrent: true,
		events:     []interface{}{target},
	}
	step := RunLodgingStep{notificationService: service}

	if err := step.waitUntil(context.Background(), target); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunLodgingStepWaitUntilRejectsUnexpectedNotificationType(t *testing.T) {
	target := time.Date(2026, time.April, 12, 15, 0, 0, 0, time.UTC)
	service := &fakeHourNotificationService{
		current:    target.Add(-time.Hour),
		hasCurrent: true,
		events:     []interface{}{"bad-type"},
	}
	step := RunLodgingStep{notificationService: service}

	err := step.waitUntil(context.Background(), target)
	if err == nil || !strings.Contains(err.Error(), "unexpected hour notification type") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunLodgingStepFinishCheckinRespondsToRequests(t *testing.T) {
	chat := &websocketfakes.Client{
		Messages: []*lodging.ChatMessage{
			{Payload: &lodging.ChatMessage_SystemNotification{SystemNotification: lodging.SystemNotification_BOOKING_CHECKING}},
			{Payload: &lodging.ChatMessage_SystemRequest{SystemRequest: lodging.SystemRequest_REQUEST_DOCUMENT}},
			{Payload: &lodging.ChatMessage_SystemRequest{SystemRequest: lodging.SystemRequest_REQUEST_BOOKING_NUMBER}},
			{Payload: &lodging.ChatMessage_SystemNotification{SystemNotification: lodging.SystemNotification_CHECK_IN_COMPLETE}},
		},
	}
	step := RunLodgingStep{flow: config.DefaultLodgingFlow()}
	cacheValue := dto.GuestJourneyCacheValue{
		PersonalInfo: &guest_registration.Guest{DocumentId: "doc-1"},
		Booking:      &dto.GuestJourneyBooking{BookingID: "booking-1"},
	}

	if err := step.finishCheckin(context.Background(), chat, cacheValue); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chat.Replies) != 2 {
		t.Fatalf("unexpected replies: %#v", chat.Replies)
	}
	if chat.Replies[0].GetShowDocument().GetDocumentId() != "doc-1" {
		t.Fatalf("unexpected document reply: %#v", chat.Replies[0])
	}
	if chat.Replies[1].GetShowBookingNumber().GetBookingId() != "booking-1" {
		t.Fatalf("unexpected booking reply: %#v", chat.Replies[1])
	}
}

func TestRunLodgingStepExpectNotificationAndRequest(t *testing.T) {
	chat := &websocketfakes.Client{
		Messages: []*lodging.ChatMessage{
			{Payload: &lodging.ChatMessage_SystemNotification{SystemNotification: lodging.SystemNotification_BREAKFAST_READY}},
			{Payload: &lodging.ChatMessage_SystemNotification{SystemNotification: lodging.SystemNotification_DINNER_READY}},
		},
	}
	step := RunLodgingStep{}

	if err := step.expectNotification(context.Background(), chat, lodging.SystemNotification_DINNER_READY); err != nil {
		t.Fatalf("unexpected notification error: %v", err)
	}

	requestChat := &websocketfakes.Client{
		Messages: []*lodging.ChatMessage{
			{Payload: &lodging.ChatMessage_SystemRequest{SystemRequest: lodging.SystemRequest_GIVE_COTTAGE_KEY}},
			{Payload: &lodging.ChatMessage_SystemRequest{SystemRequest: lodging.SystemRequest_REQUEST_COTTAGE_KEY}},
		},
	}
	msg, err := step.expectRequest(context.Background(), requestChat, lodging.SystemRequest_REQUEST_COTTAGE_KEY)
	if err != nil {
		t.Fatalf("unexpected request error: %v", err)
	}
	if msg.GetSystemRequest() != lodging.SystemRequest_REQUEST_COTTAGE_KEY {
		t.Fatalf("unexpected request message: %#v", msg)
	}
}

func TestRunLodgingStepExecuteActionPlanSendsActions(t *testing.T) {
	chat := &websocketfakes.Client{}
	step := RunLodgingStep{}
	cacheValue := dto.GuestJourneyCacheValue{
		Booking: &dto.GuestJourneyBooking{
			SelectedPeriod: &bookingdto.Period{
				Start: time.Date(2026, time.April, 12, 0, 0, 0, 0, time.UTC),
				End:   time.Date(2026, time.April, 15, 0, 0, 0, 0, time.UTC),
			},
		},
	}

	err := step.executeActionPlan(context.Background(), "stay", chat, []actionPlanStep{{
		action: lodging.GuestAction_ENTER_COTTAGE,
	}}, cacheValue)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chat.SendActions) != 1 || chat.SendActions[0] != lodging.GuestAction_ENTER_COTTAGE {
		t.Fatalf("unexpected sent actions: %#v", chat.SendActions)
	}
}

func TestRunLodgingStepBuildPlans(t *testing.T) {
	step := RunLodgingStep{flow: config.DefaultLodgingFlow()}
	cacheValue := dto.GuestJourneyCacheValue{
		Booking: &dto.GuestJourneyBooking{
			SelectedPeriod: &bookingdto.Period{
				Start: time.Date(2026, time.April, 12, 0, 0, 0, 0, time.UTC),
				End:   time.Date(2026, time.April, 15, 0, 0, 0, 0, time.UTC),
			},
		},
	}

	if got := step.expandPlan(3, []config.LodgingPlannedAction{{Action: lodging.GuestAction_WAKEUP}}); len(got) != 1 || got[0].dayOffset != 3 {
		t.Fatalf("unexpected expanded plan: %#v", got)
	}
	if got := step.buildStayActionPlan(cacheValue); len(got) <= len(step.flow.FirstDayStay) {
		t.Fatalf("expected recurring stay actions: %d", len(got))
	}
	if got := step.buildCheckoutActionPlan(cacheValue); len(got) != len(step.flow.Checkout) || got[0].dayOffset != 3 {
		t.Fatalf("unexpected checkout plan: %#v", got)
	}
}

func TestRunLodgingStepExecuteCheckoutPlanHandlesRequestAndNotification(t *testing.T) {
	chat := &websocketfakes.Client{
		Messages: []*lodging.ChatMessage{
			{Payload: &lodging.ChatMessage_SystemRequest{SystemRequest: lodging.SystemRequest_REQUEST_COTTAGE_KEY}},
			{Payload: &lodging.ChatMessage_SystemNotification{SystemNotification: lodging.SystemNotification_CHECK_OUT_COMPLETE}},
		},
	}
	step := RunLodgingStep{
		notificationService: &fakeHourNotificationService{
			current:    time.Date(2026, time.April, 20, 23, 0, 0, 0, time.UTC),
			hasCurrent: true,
		},
		flow: config.DefaultLodgingFlow(),
	}
	cacheValue := dto.GuestJourneyCacheValue{
		Booking: &dto.GuestJourneyBooking{
			SelectedPeriod: &bookingdto.Period{
				Start: time.Date(2026, time.April, 12, 0, 0, 0, 0, time.UTC),
				End:   time.Date(2026, time.April, 13, 0, 0, 0, 0, time.UTC),
			},
		},
	}

	if err := step.executeCheckoutPlan(context.Background(), chat, "key-1", cacheValue); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chat.Replies) != 1 || chat.Replies[0].GetReturnCottageKey().GetCottageKeyId() != "key-1" {
		t.Fatalf("unexpected replies: %#v", chat.Replies)
	}
	if len(chat.SendActions) != len(step.flow.Checkout) {
		t.Fatalf("unexpected actions: %#v", chat.SendActions)
	}
}

func TestRunLodgingStepExecuteSuccess(t *testing.T) {
	chat := &websocketfakes.Client{
		Messages: []*lodging.ChatMessage{
			{Payload: &lodging.ChatMessage_SystemNotification{SystemNotification: lodging.SystemNotification_BOOKING_CHECKING}},
			{Payload: &lodging.ChatMessage_SystemRequest{SystemRequest: lodging.SystemRequest_REQUEST_DOCUMENT}},
			{Payload: &lodging.ChatMessage_SystemRequest{SystemRequest: lodging.SystemRequest_REQUEST_BOOKING_NUMBER}},
			{Payload: &lodging.ChatMessage_SystemNotification{SystemNotification: lodging.SystemNotification_CHECK_IN_COMPLETE}},
			{Payload: &lodging.ChatMessage_SystemRequest{SystemRequest: lodging.SystemRequest_GIVE_COTTAGE_KEY}},
			{Payload: &lodging.ChatMessage_SystemNotification{SystemNotification: lodging.SystemNotification_DINNER_READY}},
			{Payload: &lodging.ChatMessage_SystemRequest{SystemRequest: lodging.SystemRequest_REQUEST_COTTAGE_KEY}},
			{Payload: &lodging.ChatMessage_SystemNotification{SystemNotification: lodging.SystemNotification_CHECK_OUT_COMPLETE}},
		},
	}
	factory := &fakeChatClientFactory{client: chat}
	step := RunLodgingStep{
		state:   &domain.State{GuestId: "guest-1"},
		chatURL: "ws://guest-manager/lodging/chat",
		cache: &redisfakes.Cache{LoadValue: dto.GuestJourneyCacheValue{
			PersonalInfo: &guest_registration.Guest{DocumentId: "doc-1"},
			Booking: &dto.GuestJourneyBooking{
				BookingID:       "booking-1",
				SelectedCottage: "Alps",
				SelectedPeriod: &bookingdto.Period{
					Start: time.Date(2026, time.April, 12, 0, 0, 0, 0, time.UTC),
					End:   time.Date(2026, time.April, 13, 0, 0, 0, 0, time.UTC),
				},
			},
		}},
		chatClientFactory: factory,
		notificationService: &fakeHourNotificationService{
			current:    time.Date(2026, time.April, 20, 23, 0, 0, 0, time.UTC),
			hasCurrent: true,
		},
		flow: config.DefaultLodgingFlow(),
	}

	if err := step.Execute(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if factory.url != "ws://guest-manager/lodging/chat" {
		t.Fatalf("unexpected factory url: %q", factory.url)
	}
	if len(chat.Replies) != 4 {
		t.Fatalf("unexpected reply count: %d", len(chat.Replies))
	}
	if len(chat.SendActions) == 0 {
		t.Fatal("expected actions to be sent")
	}
}

func TestGuestResponseBuilders(t *testing.T) {
	if got := guestResponseShowDocument("doc-1").GetShowDocument().GetDocumentId(); got != "doc-1" {
		t.Fatalf("unexpected document id: %q", got)
	}
	if got := guestResponseShowBookingNumber("booking-1").GetShowBookingNumber().GetBookingId(); got != "booking-1" {
		t.Fatalf("unexpected booking id: %q", got)
	}
	if got := guestResponseReceiveCottageKey("key-1").GetReceiveCottageKey().GetCottageKeyId(); got != "key-1" {
		t.Fatalf("unexpected received key id: %q", got)
	}
	if got := guestResponseReturnCottageKey("key-1").GetReturnCottageKey().GetCottageKeyId(); got != "key-1" {
		t.Fatalf("unexpected returned key id: %q", got)
	}
}

func TestRunLodgingStepExpectNotificationReturnsChatError(t *testing.T) {
	step := RunLodgingStep{}
	err := step.expectNotification(context.Background(), &websocketfakes.Client{WaitErr: errors.New("chat failed")}, lodging.SystemNotification_DINNER_READY)
	if err == nil || !strings.Contains(err.Error(), "chat failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

var (
	_ port.Cache                       = (*redisfakes.Cache)(nil)
	_ port.LodgingChatClient           = (*websocketfakes.Client)(nil)
	_ services.HourNotificationService = (*fakeHourNotificationService)(nil)
)
