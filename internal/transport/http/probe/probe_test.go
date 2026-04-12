package probe

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/Kenji-Uema/guestSimulator/internal/infra/mq"
	"github.com/go-resty/resty/v2"
)

func TestHealthHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	HealthHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status code: %d", rec.Code)
	}
}

func TestPingHealthz(t *testing.T) {
	successClient := resty.New().
		SetBaseURL("http://example.test").
		SetTransport(roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if r.URL.Path != "/healthz" {
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}
			return response(r, http.StatusOK, []byte("ok")), nil
		}))

	if err := pingHealthz(successClient, context.Background(), "CottageManager"); err != nil {
		t.Fatalf("unexpected success error: %v", err)
	}

	failureClient := resty.New().
		SetBaseURL("http://example.test").
		SetTransport(roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return response(r, http.StatusServiceUnavailable, []byte("down")), nil
		}))

	err := pingHealthz(failureClient, context.Background(), "GuestManager")
	if err == nil || !strings.Contains(err.Error(), "GuestManager healthz ping failed") {
		t.Fatalf("unexpected failure error: %v", err)
	}
}

func TestReadinessHandlerReturnsUnavailableWhenRabbitConnectionIsClosed(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()

	ReadinessHandler(&mq.RabbitMqConnection{}, nil, nil, nil).ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("unexpected status code: %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "RabbitMQ connection is not open") {
		t.Fatalf("unexpected response body: %q", rec.Body.String())
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func response(r *http.Request, status int, body []byte) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     strconv.Itoa(status) + " " + http.StatusText(status),
		Header:     make(http.Header),
		Body:       io.NopCloser(bytes.NewReader(body)),
		Request:    r,
	}
}
