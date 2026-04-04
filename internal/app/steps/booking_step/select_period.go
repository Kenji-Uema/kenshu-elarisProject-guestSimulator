package booking_step

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"log/slog"
	"sort"
	"time"

	"github.com/Kenji-Uema/guestSimulator/internal/app/journeyctx"
	"github.com/Kenji-Uema/guestSimulator/internal/app/steps"
	"github.com/Kenji-Uema/guestSimulator/internal/domain"
	redisc "github.com/Kenji-Uema/guestSimulator/internal/infra/redis"
	"github.com/Kenji-Uema/guestSimulator/internal/infra/telemetry"
	"github.com/Kenji-Uema/guestSimulator/internal/transport/grpc"
	"github.com/go-resty/resty/v2"
)

var numberOfNights = []int{3, 5, 7, 10, 14}
var searchWindows = []int{3, 5, 7, 10, 14, 21, 30}

var ErrNoSuitablePeriod = errors.New("no suitable period found")

type SelectPeriodStep struct {
	clock  *grpc.Emu
	client *resty.Client
	state  *domain.State
	redis  *redisc.Redis
}

type stayCandidate struct {
	cottageName string
	period      *domain.Period
	distance    time.Duration
	nights      int
}

func NewSelectPeriodStep(state *domain.State, clock *grpc.Emu, client *resty.Client, redis *redisc.Redis) steps.Step {
	return &SelectPeriodStep{clock: clock, client: client, state: state, redis: redis}
}

func (s SelectPeriodStep) Name() string {
	return "SelectPeriodStep"
}

func (s SelectPeriodStep) Validate() error {
	if s.state == nil {
		return fmt.Errorf("invalid state, state is nil")
	}
	if s.redis == nil {
		return fmt.Errorf("invalid redis client")
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

	cacheValue, err := journeyctx.Load(ctx, s.redis, s.state)
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

	for _, windowDays := range searchWindows {
		selectedCottage, selectedPeriod, ok, err := s.selectNearestStay(ctx, *now, windowDays, nightOptions)
		if err != nil {
			return err
		}
		if ok {
			cacheValue.Booking.SelectedCottage = selectedCottage
			cacheValue.Booking.SelectedPeriod = selectedPeriod
			if err := journeyctx.Save(ctx, s.redis, s.state, cacheValue); err != nil {
				return err
			}
			slog.InfoContext(ctx, "state updated, stay period selected", "selectedCottage", selectedCottage, "selectedPeriod", selectedPeriod)
			return nil
		}
	}

	slog.WarnContext(ctx, "No suitable period found")
	return ErrNoSuitablePeriod
}

func (s SelectPeriodStep) selectNearestStay(ctx context.Context, now time.Time, windowDays int, nightOptions []int) (string, *domain.Period, bool, error) {
	candidates := make([]stayCandidate, 0, len(s.state.CottageNames))

	for _, cottageName := range s.state.CottageNames {
		availablePeriods, err := s.loadAvailablePeriods(ctx, cottageName, now, windowDays)
		if err != nil {
			return "", nil, false, err
		}

		for _, nights := range nightOptions {
			selectedPeriod, ok := pickNearestSuitablePeriod(now, availablePeriods, nights)
			if !ok {
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

	if len(candidates) == 0 {
		return "", nil, false, nil
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

	selected := candidates[s.pickCandidateIndex(candidates)]
	return selected.cottageName, selected.period, true, nil
}

func (s SelectPeriodStep) pickCandidateIndex(candidates []stayCandidate) int {
	if len(candidates) == 0 {
		return 0
	}

	topN := len(candidates)
	if topN > 5 {
		topN = 5
	}
	if topN == 1 {
		return 0
	}

	hasher := fnv.New32a()
	_, _ = hasher.Write([]byte(s.state.GuestId))
	return int(hasher.Sum32() % uint32(topN))
}

func (s SelectPeriodStep) loadAvailablePeriods(ctx context.Context, cottageName string, now time.Time, windowDays int) ([]domain.Period, error) {
	from := now
	to := now.AddDate(0, 0, windowDays)

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

	var availablePeriodDTO domain.AvailablePeriodDTO
	if err := json.Unmarshal(resp.Body(), &availablePeriodDTO); err != nil {
		return nil, err
	}

	return availablePeriodDTO.ToPeriods(), nil
}

func pickNearestSuitablePeriod(now time.Time, availablePeriods []domain.Period, nights int) (*domain.Period, bool) {
	var selected *domain.Period
	var selectedDistance time.Duration

	searchStart := startOfUTCDay(now).AddDate(0, 0, 1)

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
			candidate := domain.Period{
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
