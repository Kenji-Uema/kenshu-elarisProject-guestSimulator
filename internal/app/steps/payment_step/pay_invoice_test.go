package payment_step

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Kenji-Uema/guestSimulator/internal/domain"
	"github.com/Kenji-Uema/guestSimulator/internal/domain/dto"
	bookingdto "github.com/Kenji-Uema/guestSimulator/internal/domain/dto/booking"
	"github.com/Kenji-Uema/guestSimulator/internal/domain/dto/guest_registration"
	redisfakes "github.com/Kenji-Uema/guestSimulator/internal/infra/redis/fakes"
	"github.com/Kenji-Uema/guestSimulator/internal/port"
	"github.com/go-resty/resty/v2"
)

func TestPayInvoiceStepValidateRejectsMissingState(t *testing.T) {
	err := PayInvoiceStep{}.Validate()
	if err == nil || !strings.Contains(err.Error(), "state is nil") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNewPayInvoiceStepReturnsNamedStep(t *testing.T) {
	step := NewPayInvoiceStep(&domain.State{}, newTestClient(func(r *http.Request) (*http.Response, error) {
		t.Fatalf("unexpected request: %v", r.URL)
		return nil, nil
	}), &redisfakes.Cache{})
	if step == nil {
		t.Fatal("expected step")
	}
	if step.Name() != "PayInvoiceStep" {
		t.Fatalf("unexpected step name: %q", step.Name())
	}
}

func TestPayInvoiceStepExecuteSavesReceiptNumber(t *testing.T) {
	cache := &redisfakes.Cache{
		LoadValue: dto.GuestJourneyCacheValue{
			PersonalInfo: &guest_registration.Guest{
				GivenNames: "Guest",
				Surname:    "Test",
			},
			Booking: &dto.GuestJourneyBooking{
				BookingID:       "booking-1",
				SelectedCottage: "Alps",
				SelectedPeriod: &bookingdto.Period{
					Start: time.Date(2026, time.April, 12, 0, 0, 0, 0, time.UTC),
					End:   time.Date(2026, time.April, 15, 0, 0, 0, 0, time.UTC),
				},
			},
			Invoice: &dto.GuestJourneyInvoice{InvoiceNumber: "invoice-1"},
		},
	}
	step := PayInvoiceStep{
		paymentClient: newTestClient(func(r *http.Request) (*http.Response, error) {
			if r.URL.Path != "/v1/payments/invoice/invoice-1" {
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}
			var req dto.PayWithCardRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode request: %v", err)
			}
			if req.HolderName != "Guest Test" {
				t.Fatalf("unexpected holder name: %q", req.HolderName)
			}
			return jsonResponse(r, http.StatusOK, dto.PayWithCardResponse{
				ReceiptNumber: "receipt-1",
				InvoiceNumber: "invoice-1",
				Status:        "PAYMENT_STATUS_SUCCEEDED",
			}), nil
		}),
		cache: cache,
		state: &domain.State{GuestId: "guest-1"},
	}

	if err := step.Execute(context.Background()); err != nil {
		t.Fatalf("unexpected execute error: %v", err)
	}
	if cache.SavedValue.Invoice == nil || cache.SavedValue.Invoice.ReceiptNumber != "receipt-1" {
		t.Fatalf("unexpected invoice cache: %#v", cache.SavedValue.Invoice)
	}
}

func TestPayInvoiceStepExecuteRejectsUnexpectedStatus(t *testing.T) {
	cache := &redisfakes.Cache{
		LoadValue: dto.GuestJourneyCacheValue{
			PersonalInfo: &guest_registration.Guest{
				GivenNames: "Guest",
				Surname:    "Test",
			},
			Booking: &dto.GuestJourneyBooking{
				BookingID:       "booking-1",
				SelectedCottage: "Alps",
				SelectedPeriod: &bookingdto.Period{
					Start: time.Date(2026, time.April, 12, 0, 0, 0, 0, time.UTC),
					End:   time.Date(2026, time.April, 15, 0, 0, 0, 0, time.UTC),
				},
			},
			Invoice: &dto.GuestJourneyInvoice{InvoiceNumber: "invoice-1"},
		},
	}
	step := PayInvoiceStep{
		paymentClient: newTestClient(func(r *http.Request) (*http.Response, error) {
			return jsonResponse(r, http.StatusOK, dto.PayWithCardResponse{
				ReceiptNumber: "receipt-1",
				Status:        "PAYMENT_STATUS_FAILED",
			}), nil
		}),
		cache: cache,
		state: &domain.State{GuestId: "guest-1"},
	}

	err := step.Execute(context.Background())
	if err == nil || !strings.Contains(err.Error(), "returned status PAYMENT_STATUS_FAILED") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPayInvoiceStepExecuteRejectsMissingInvoiceContext(t *testing.T) {
	step := PayInvoiceStep{
		paymentClient: newTestClient(func(r *http.Request) (*http.Response, error) {
			t.Fatalf("unexpected request: %v", r.URL)
			return nil, nil
		}),
		cache: &redisfakes.Cache{
			LoadValue: dto.GuestJourneyCacheValue{
				PersonalInfo: &guest_registration.Guest{GivenNames: "Guest"},
				Booking: &dto.GuestJourneyBooking{
					BookingID:       "booking-1",
					SelectedCottage: "Alps",
					SelectedPeriod: &bookingdto.Period{
						Start: time.Date(2026, time.April, 12, 0, 0, 0, 0, time.UTC),
						End:   time.Date(2026, time.April, 15, 0, 0, 0, 0, time.UTC),
					},
				},
			},
		},
		state: &domain.State{GuestId: "guest-1"},
	}

	err := step.Execute(context.Background())
	if err == nil || !strings.Contains(err.Error(), "invalid cached invoice context") {
		t.Fatalf("unexpected error: %v", err)
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
