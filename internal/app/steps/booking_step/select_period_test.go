package booking_step

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Kenji-Uema/guestSimulator/internal/domain"
	"github.com/Kenji-Uema/guestSimulator/internal/domain/dto/booking"
	redisfakes "github.com/Kenji-Uema/guestSimulator/internal/infra/redis/fakes"
	"github.com/go-resty/resty/v2"
)

func TestNewSelectPeriodStepReturnsNamedStep(t *testing.T) {
	step := NewSelectPeriodStep(&domain.State{}, nil, resty.New(), &redisfakes.Cache{})

	if step == nil {
		t.Fatal("expected step")
	}
	if step.Name() != "SelectPeriodStep" {
		t.Fatalf("unexpected step name: %q", step.Name())
	}
}

func TestSelectPeriodStepValidateRejectsMissingCache(t *testing.T) {
	err := SelectPeriodStep{state: &domain.State{}}.Validate()
	if err == nil || !strings.Contains(err.Error(), "invalid guest journey cache") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSelectPeriodLoadAvailablePeriodsBuildsExpectedQuery(t *testing.T) {
	now := time.Date(2026, time.April, 7, 10, 45, 0, 0, time.UTC)
	step := SelectPeriodStep{
		client: newTestClient(func(r *http.Request) (*http.Response, error) {
			if r.URL.Query().Get("from") != "2026-04-12" {
				t.Fatalf("unexpected from query: %q", r.URL.Query().Get("from"))
			}
			if r.URL.Query().Get("to") != "2026-04-15" {
				t.Fatalf("unexpected to query: %q", r.URL.Query().Get("to"))
			}
			return jsonResponse(r, http.StatusOK, booking.AvailablePeriodDTO{
				Name: "Alps",
				Periods: []booking.PeriodDTO{{
					CheckIn:  time.Date(2026, time.April, 12, 0, 0, 0, 0, time.UTC),
					CheckOut: time.Date(2026, time.April, 15, 0, 0, 0, 0, time.UTC),
				}},
			}), nil
		}),
	}

	periods, err := step.loadAvailablePeriods(context.Background(), "Alps", now, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(periods) != 1 {
		t.Fatalf("unexpected periods: %#v", periods)
	}
}
