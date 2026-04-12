package booking_step

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Kenji-Uema/guestSimulator/internal/domain"
	"github.com/Kenji-Uema/guestSimulator/internal/domain/dto"
	bookingdto "github.com/Kenji-Uema/guestSimulator/internal/domain/dto/booking"
	"github.com/Kenji-Uema/guestSimulator/internal/domain/dto/guest_registration"
	clockfakes "github.com/Kenji-Uema/guestSimulator/internal/infra/clock/fakes"
	redisfakes "github.com/Kenji-Uema/guestSimulator/internal/infra/redis/fakes"
	"github.com/Kenji-Uema/guestSimulator/internal/port"
	"github.com/go-resty/resty/v2"
)

func TestListCottagesStepExecuteStoresCottageNames(t *testing.T) {
	state := &domain.State{}
	step := ListCottagesStep{
		client: newTestClient(func(r *http.Request) (*http.Response, error) {
			if r.URL.Path != "/cottages" {
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}
			return jsonResponse(r, http.StatusOK, []bookingdto.Cottage{
				{Name: "Alps"},
				{Name: "Lagoon"},
			}), nil
		}),
		state: state,
	}

	if err := step.Execute(context.Background()); err != nil {
		t.Fatalf("unexpected execute error: %v", err)
	}
	if !reflect.DeepEqual(state.CottageNames, []string{"Alps", "Lagoon"}) {
		t.Fatalf("unexpected cottage names: %#v", state.CottageNames)
	}
}

func TestSelectCottageStepValidateAndExecute(t *testing.T) {
	cache := &redisfakes.Cache{LoadValue: dto.GuestJourneyCacheValue{}}
	state := &domain.State{CottageNames: []string{"Alps"}}
	step := SelectCottageStep{state: state, cache: cache}

	if err := step.Validate(); err != nil {
		t.Fatalf("unexpected validate error: %v", err)
	}
	if err := step.Execute(context.Background()); err != nil {
		t.Fatalf("unexpected execute error: %v", err)
	}
	if cache.SavedValue.Booking == nil || cache.SavedValue.Booking.SelectedCottage != "Alps" {
		t.Fatalf("unexpected saved booking: %#v", cache.SavedValue.Booking)
	}
}

func TestPickNearestSuitablePeriod(t *testing.T) {
	now := time.Date(2026, time.April, 7, 10, 45, 0, 0, time.UTC)
	searchStart := startOfUTCDay(now).AddDate(0, 0, minSearchLeadDays)
	selected, ok := pickNearestSuitablePeriod(now, []bookingdto.Period{
		{
			Start: searchStart.AddDate(0, 0, -2),
			End:   searchStart.AddDate(0, 0, 10),
		},
	}, 3)
	if !ok || selected == nil {
		t.Fatal("expected period to be selected")
	}
	if !selected.Start.Equal(searchStart) {
		t.Fatalf("unexpected selected start: %s", selected.Start)
	}
	if !selected.End.Equal(searchStart.AddDate(0, 0, 3)) {
		t.Fatalf("unexpected selected end: %s", selected.End)
	}
}

func TestUniqueStayCandidatesRemovesDuplicates(t *testing.T) {
	period := &bookingdto.Period{
		Start: time.Date(2026, time.April, 12, 0, 0, 0, 0, time.UTC),
		End:   time.Date(2026, time.April, 15, 0, 0, 0, 0, time.UTC),
	}

	candidates := uniqueStayCandidates([]stayCandidate{
		{cottageName: "Alps", period: period},
		{cottageName: "Alps", period: period},
		{cottageName: "Lagoon", period: period},
	})

	if len(candidates) != 2 {
		t.Fatalf("unexpected candidates: %#v", candidates)
	}
}

func TestSelectPeriodExecuteChoosesAndSavesNearestPeriod(t *testing.T) {
	now := time.Date(2026, time.April, 7, 10, 45, 0, 0, time.UTC)
	cache := &redisfakes.Cache{
		LoadValue: dto.GuestJourneyCacheValue{
			Booking: &dto.GuestJourneyBooking{},
		},
	}
	state := &domain.State{CottageNames: []string{"Alps"}}
	step := SelectPeriodStep{
		clock: clockfakes.Clock{NowValue: now},
		client: newTestClient(func(r *http.Request) (*http.Response, error) {
			if !strings.HasPrefix(r.URL.Path, "/cottage/Alps/available-dates") {
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}
			return jsonResponse(r, http.StatusOK, bookingdto.AvailablePeriodDTO{
				Name: "Alps",
				Periods: []bookingdto.PeriodDTO{{
					CheckIn:  now.AddDate(0, 0, 4),
					CheckOut: now.AddDate(0, 0, 20),
				}},
			}), nil
		}),
		state: state,
		cache: cache,
	}

	if err := step.Execute(context.Background()); err != nil {
		t.Fatalf("unexpected execute error: %v", err)
	}

	expectedStart := startOfUTCDay(now).AddDate(0, 0, minSearchLeadDays)
	expectedEnd := expectedStart.AddDate(0, 0, 3)
	if cache.SavedValue.Booking == nil {
		t.Fatal("expected booking to be saved")
	}
	if cache.SavedValue.Booking.SelectedCottage != "Alps" {
		t.Fatalf("unexpected selected cottage: %q", cache.SavedValue.Booking.SelectedCottage)
	}
	if !cache.SavedValue.Booking.SelectedPeriod.Start.Equal(expectedStart) {
		t.Fatalf("unexpected selected period start: %s", cache.SavedValue.Booking.SelectedPeriod.Start)
	}
	if !cache.SavedValue.Booking.SelectedPeriod.End.Equal(expectedEnd) {
		t.Fatalf("unexpected selected period end: %s", cache.SavedValue.Booking.SelectedPeriod.End)
	}
}

func TestBookCottageStepExecuteCreatesBooking(t *testing.T) {
	cache := &redisfakes.Cache{
		LoadValue: dto.GuestJourneyCacheValue{
			PersonalInfo: &guest_registration.Guest{
				GivenNames:     "Guest",
				Surname:        "Test",
				Email:          "guest@test.com",
				DocumentId:     "123",
				BillingAddress: "Main Street",
			},
			Booking: &dto.GuestJourneyBooking{
				SelectedCottage: "Alps",
				SelectedPeriod: &bookingdto.Period{
					Start: time.Date(2026, time.April, 12, 0, 0, 0, 0, time.UTC),
					End:   time.Date(2026, time.April, 15, 0, 0, 0, 0, time.UTC),
				},
			},
		},
	}
	step := BookCottageStep{
		client: newTestClient(func(r *http.Request) (*http.Response, error) {
			if r.URL.Path != "/cottage/Alps/booking" {
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}
			var req bookingdto.Request
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode request: %v", err)
			}
			if req.GuestId != "guest-1" || req.GuestEmail != "guest@test.com" {
				t.Fatalf("unexpected request body: %#v", req)
			}
			return jsonResponse(r, http.StatusOK, bookingdto.Confirmation{Id: "booking-1"}), nil
		}),
		state: &domain.State{GuestId: "guest-1"},
		cache: cache,
	}

	if err := step.Execute(context.Background()); err != nil {
		t.Fatalf("unexpected execute error: %v", err)
	}
	if cache.SavedValue.Booking == nil || cache.SavedValue.Booking.BookingID != "booking-1" {
		t.Fatalf("unexpected saved booking: %#v", cache.SavedValue.Booking)
	}
}

func TestBookCottageStepValidateRejectsMissingGuestID(t *testing.T) {
	err := BookCottageStep{state: &domain.State{}, cache: &redisfakes.Cache{}}.Validate()
	if err == nil || !strings.Contains(err.Error(), "guestId is empty") {
		t.Fatalf("unexpected error: %v", err)
	}
}

var (
	_ port.Cache = (*redisfakes.Cache)(nil)
	_ port.Clock = clockfakes.Clock{}
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func newTestClient(fn roundTripFunc) *resty.Client {
	return resty.New().
		SetBaseURL("http://example.test").
		SetTransport(fn)
}

func jsonResponse(r *http.Request, status int, payload any) *http.Response {
	body, err := json.Marshal(payload)
	if err != nil {
		panic(err)
	}

	return &http.Response{
		StatusCode: status,
		Status:     fmtStatus(status),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(bytes.NewReader(body)),
		Request:    r,
	}
}

func fmtStatus(status int) string {
	return strconv.Itoa(status) + " " + http.StatusText(status)
}
