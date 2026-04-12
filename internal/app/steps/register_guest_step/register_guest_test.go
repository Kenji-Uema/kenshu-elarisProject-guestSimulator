package register_guest_step

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/Kenji-Uema/guestSimulator/internal/domain"
	"github.com/Kenji-Uema/guestSimulator/internal/domain/dto"
	"github.com/Kenji-Uema/guestSimulator/internal/domain/dto/guest_registration"
	redisfakes "github.com/Kenji-Uema/guestSimulator/internal/infra/redis/fakes"
	"github.com/Kenji-Uema/guestSimulator/internal/port"
	"github.com/go-resty/resty/v2"
)

func TestRegisterGuestStepValidateRejectsMissingDependencies(t *testing.T) {
	err := RegisterGuestStep{}.Validate()
	if err == nil || !strings.Contains(err.Error(), "invalid guest client") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNewRegisterGuestStepReturnsNamedStep(t *testing.T) {
	step := NewRegisterGuestStep(newTestClient(func(r *http.Request) (*http.Response, error) {
		t.Fatalf("unexpected request: %v", r.URL)
		return nil, nil
	}), &redisfakes.Cache{}, &domain.State{})
	if step == nil {
		t.Fatal("expected step")
	}
	if step.Name() != "RegisterGuestStep" {
		t.Fatalf("unexpected step name: %q", step.Name())
	}
}

func TestRegisterGuestStepExecuteRegistersAndSavesGuest(t *testing.T) {
	cache := &redisfakes.Cache{
		LoadValue: dto.GuestJourneyCacheValue{
			PersonalInfo: &guest_registration.Guest{
				Email:          "guest@test.com",
				GivenNames:     "Guest",
				Surname:        "Test",
				DocumentId:     "123",
				BillingAddress: "Main Street",
			},
		},
	}
	state := &domain.State{}
	step := RegisterGuestStep{
		client: newTestClient(func(r *http.Request) (*http.Response, error) {
			if r.URL.Path != "/guest" {
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}
			var guest guest_registration.Guest
			if err := json.NewDecoder(r.Body).Decode(&guest); err != nil {
				t.Fatalf("decode request: %v", err)
			}
			if guest.Email != "guest@test.com" {
				t.Fatalf("unexpected guest: %#v", guest)
			}
			return jsonResponse(r, http.StatusOK, "guest-1"), nil
		}),
		cache: cache,
		state: state,
	}

	if err := step.Execute(context.Background()); err != nil {
		t.Fatalf("unexpected execute error: %v", err)
	}
	if state.GuestId != "guest-1" {
		t.Fatalf("unexpected guest id: %q", state.GuestId)
	}
	if cache.SavedValue.GuestID != "guest-1" {
		t.Fatalf("unexpected cached guest id: %q", cache.SavedValue.GuestID)
	}
}

func TestRegisterGuestStepExecuteRejectsMissingPersonalInfo(t *testing.T) {
	step := RegisterGuestStep{
		client: newTestClient(func(r *http.Request) (*http.Response, error) {
			t.Fatalf("unexpected request: %v", r.URL)
			return nil, nil
		}),
		cache: &redisfakes.Cache{LoadValue: dto.GuestJourneyCacheValue{}},
		state: &domain.State{},
	}

	err := step.Execute(context.Background())
	if err == nil || !strings.Contains(err.Error(), "invalid cached guest context") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRetrieveGuestStepValidateAndExecute(t *testing.T) {
	step := RetrieveGuestStep{
		client: newTestClient(func(r *http.Request) (*http.Response, error) {
			if r.URL.Path != "/guest/guest-1" {
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Status:     fmtStatus(http.StatusOK),
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader("ok")),
				Request:    r,
			}, nil
		}),
		state: &domain.State{GuestId: "guest-1"},
	}

	if err := step.Validate(); err != nil {
		t.Fatalf("unexpected validate error: %v", err)
	}
	if err := step.Execute(context.Background()); err != nil {
		t.Fatalf("unexpected execute error: %v", err)
	}
}

var _ port.Cache = (*redisfakes.Cache)(nil)

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
