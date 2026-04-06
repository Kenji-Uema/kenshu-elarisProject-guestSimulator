package booking_step

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"time"

	"github.com/Kenji-Uema/guestSimulator/internal/app/steps"
	"github.com/Kenji-Uema/guestSimulator/internal/domain"
	"github.com/Kenji-Uema/guestSimulator/internal/domain/dto/booking"
	"github.com/Kenji-Uema/guestSimulator/internal/infra/telemetry"
	"github.com/Kenji-Uema/guestSimulator/internal/port"
	"github.com/go-resty/resty/v2"
)

var numberOfNights = []int{3, 5, 7, 10, 14}
var searchWindows = []int{3, 5, 7, 10, 14, 21, 30}

const maxPeriodAttempts = 5
const minSearchLeadDays = 5

var ErrNoSuitablePeriod = errors.New("no suitable period found")

type SelectPeriodStep struct {
	clock  port.Clock
	client *resty.Client
	state  *domain.State
	cache  port.Cache
}

type stayCandidate struct {
	cottageName string
	period      *booking.Period
	distance    time.Duration
	nights      int
}

func NewSelectPeriodStep(state *domain.State, clock port.Clock, client *resty.Client, cache port.Cache) steps.Step {
	return &SelectPeriodStep{clock: clock, client: client, state: state, cache: cache}
}

func (s SelectPeriodStep) Name() string {
	return "SelectPeriodStep"
}

func (s SelectPeriodStep) Validate() error {
	if s.state == nil {
		return fmt.Errorf("invalid state, state is nil")
	}
	if s.cache == nil {
		return fmt.Errorf("invalid guest journey cache")
	}

	return nil
}

func (s SelectPeriodStep) Execute(ctx context.Context) error {
	ctx, span := telemetry.Tracer.Start(ctx, "SelectPeriodStep")
	defer span.End()

	slog.InfoContext(ctx, "User selects a period of time")

	now, err := s.clock.Now(ctx)
	if err != nil {
		return err
	}

	cacheValue, err := s.cache.Load(ctx, s.state)
	if err != nil {
		return err
	}
	if cacheValue.Booking == nil {
		return fmt.Errorf("invalid cached booking context, booking is nil")
	}
	if len(s.state.CottageNames) == 0 {
		return fmt.Errorf("invalid state, cottageNames is empty")
	}

	nightOptions := append([]int(nil), numberOfNights...)
	sort.Ints(nightOptions)

	candidates, err := s.collectStayCandidates(ctx, *now, nightOptions)
	if err != nil {
		return err
	}

	if len(candidates) == 0 {
		slog.WarnContext(ctx, "No suitable period found")
		return ErrNoSuitablePeriod
	}

	if len(candidates) > maxPeriodAttempts {
		candidates = candidates[:maxPeriodAttempts]
	}

	selected := candidates[0]
	cacheValue.Booking.SelectedCottage = selected.cottageName
	cacheValue.Booking.SelectedPeriod = selected.period
	if err := s.cache.Save(ctx, s.state, cacheValue); err != nil {
		return err
	}
	slog.InfoContext(ctx, "state updated, stay period selected", "selectedCottage", selected.cottageName, "selectedPeriod", selected.period, "candidateAttempts", len(candidates))
	return nil
}

func (s SelectPeriodStep) collectStayCandidates(ctx context.Context, now time.Time, nightOptions []int) ([]stayCandidate, error) {
	candidates := make([]stayCandidate, 0, len(s.state.CottageNames))
	for _, cottageName := range s.state.CottageNames {
		for _, windowDays := range searchWindows {
			availablePeriods, err := s.loadAvailablePeriods(ctx, cottageName, now, windowDays)
			if err != nil {
				return nil, err
			}

			for _, nights := range nightOptions {
				selectedPeriod, ok := pickNearestSuitablePeriod(now, availablePeriods, nights)
				if !ok || selectedPeriod == nil {
					continue
				}

				candidates = append(candidates, stayCandidate{
					cottageName: cottageName,
					period:      selectedPeriod,
					distance:    selectedPeriod.Start.Sub(now),
					nights:      nights,
				})
				break
			}
		}
	}

	if len(candidates) == 0 {
		return nil, nil
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].distance != candidates[j].distance {
			return candidates[i].distance < candidates[j].distance
		}
		if candidates[i].nights != candidates[j].nights {
			return candidates[i].nights < candidates[j].nights
		}
		if candidates[i].period.Start != candidates[j].period.Start {
			return candidates[i].period.Start.Before(candidates[j].period.Start)
		}
		return candidates[i].cottageName < candidates[j].cottageName
	})

	return uniqueStayCandidates(candidates), nil
}

func (s SelectPeriodStep) loadAvailablePeriods(ctx context.Context, cottageName string, now time.Time, windowDays int) ([]booking.Period, error) {
	from := startOfUTCDay(now).AddDate(0, 0, minSearchLeadDays)
	to := from.AddDate(0, 0, windowDays)

	resp, err := s.client.R().
		SetContext(ctx).
		SetQueryParams(map[string]string{"to": to.Format("2006-01-02"), "from": from.Format("2006-01-02")}).
		Get(fmt.Sprintf("/cottage/%s/available-dates", cottageName))
	if err != nil {
		return nil, err
	}

	if resp.IsError() {
		return nil, fmt.Errorf("error: %s", resp.Status())
	}

	var availablePeriodDTO booking.AvailablePeriodDTO
	if err := json.Unmarshal(resp.Body(), &availablePeriodDTO); err != nil {
		return nil, err
	}

	return availablePeriodDTO.ToPeriods(), nil
}

func pickNearestSuitablePeriod(now time.Time, availablePeriods []booking.Period, nights int) (*booking.Period, bool) {
	var selected *booking.Period
	var selectedDistance time.Duration

	searchStart := startOfUTCDay(now).AddDate(0, 0, minSearchLeadDays)

	for _, period := range availablePeriods {
		candidateStart := startOfUTCDay(period.Start)
		if candidateStart.Before(searchStart) {
			candidateStart = searchStart
		}

		candidateEnd := candidateStart.AddDate(0, 0, nights)
		periodEnd := startOfUTCDay(period.End)
		if candidateEnd.After(periodEnd) {
			continue
		}

		distance := candidateStart.Sub(searchStart)
		if selected == nil || distance < selectedDistance {
			candidate := booking.Period{
				Start: candidateStart,
				End:   candidateEnd,
			}
			selected = &candidate
			selectedDistance = distance
		}
	}

	return selected, selected != nil
}

func startOfUTCDay(t time.Time) time.Time {
	return time.Date(t.UTC().Year(), t.UTC().Month(), t.UTC().Day(), 0, 0, 0, 0, time.UTC)
}

func uniqueStayCandidates(candidates []stayCandidate) []stayCandidate {
	unique := make([]stayCandidate, 0, len(candidates))
	seen := make(map[string]struct{}, len(candidates))

	for _, candidate := range candidates {
		key := fmt.Sprintf("%s|%s|%s", candidate.cottageName, candidate.period.Start.UTC().Format(time.RFC3339), candidate.period.End.UTC().Format(time.RFC3339))
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		unique = append(unique, candidate)
	}

	return unique
}
