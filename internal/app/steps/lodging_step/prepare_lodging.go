package lodging_step

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/Kenji-Uema/guestSimulator/internal/app/steps"
	"github.com/Kenji-Uema/guestSimulator/internal/domain"
	"github.com/Kenji-Uema/guestSimulator/internal/domain/dto"
	"github.com/Kenji-Uema/guestSimulator/internal/infra/telemetry"
	"github.com/Kenji-Uema/guestSimulator/internal/port"
	"github.com/go-resty/resty/v2"
)

type PrepareLodgingStep struct {
	clock         port.Clock
	cottageClient *resty.Client
	state         *domain.State
	cache         port.Cache
}

func NewPrepareLodgingStep(state *domain.State, clock port.Clock, cottageClient *resty.Client, cache port.Cache) steps.Step {
	return &PrepareLodgingStep{clock: clock, cottageClient: cottageClient, state: state, cache: cache}
}

func (s PrepareLodgingStep) Name() string {
	return "PrepareLodgingStep"
}

func (s PrepareLodgingStep) Validate() error {
	if s.state == nil {
		return fmt.Errorf("invalid state, state is nil")
	}
	if s.cache == nil {
		return fmt.Errorf("invalid guest journey cache")
	}

	return nil
}

func (s PrepareLodgingStep) Execute(ctx context.Context) error {
	ctx, span := telemetry.Tracer.Start(ctx, "PrepareLodgingStep")
	defer span.End()

	now, err := s.clock.Now(ctx)
	if err != nil {
		return err
	}

	cottageNames, err := s.listCottages(ctx)
	if err != nil {
		return err
	}

	checkIn := time.Date(now.UTC().Year(), now.UTC().Month(), now.UTC().Day(), 0, 0, 0, 0, time.UTC)
	checkOut := checkIn.AddDate(0, 0, 1)

	for _, cottageName := range cottageNames {
		ok, err := s.isAvailableForLodging(ctx, cottageName, checkIn, checkOut)
		if err != nil {
			return err
		}
		if !ok {
			continue
		}

		cacheValue, err := s.cache.Load(ctx, s.state)
		if err != nil {
			return err
		}
		if cacheValue.Booking == nil {
			cacheValue.Booking = &dto.GuestJourneyBooking{}
		}
		cacheValue.Booking.SelectedCottage = cottageName
		cacheValue.Booking.SelectedPeriod = &domain.Period{
			Start: checkIn,
			End:   checkOut,
		}
		if err := s.cache.Save(ctx, s.state, cacheValue); err != nil {
			return err
		}

		slog.InfoContext(ctx, "prepared lodging booking window", "cottage", cottageName, "checkIn", checkIn, "checkOut", checkOut)
		return nil
	}

	return fmt.Errorf("no cottage available for %s to %s", checkIn.Format("2006-01-02"), checkOut.Format("2006-01-02"))
}

func (s PrepareLodgingStep) listCottages(ctx context.Context) ([]string, error) {
	resp, err := s.cottageClient.R().
		SetContext(ctx).
		Get("/cottages")
	if err != nil {
		return nil, err
	}
	if resp.IsError() {
		return nil, fmt.Errorf("error: %s", resp.Status())
	}

	var cottageNames []string
	if err := json.Unmarshal(resp.Body(), &cottageNames); err != nil {
		return nil, err
	}

	return cottageNames, nil
}

func (s PrepareLodgingStep) isAvailableForLodging(ctx context.Context, cottageName string, checkIn time.Time, checkOut time.Time) (bool, error) {
	resp, err := s.cottageClient.R().
		SetContext(ctx).
		SetQueryParams(map[string]string{
			"from": checkIn.Format("2006-01-02"),
			"to":   checkOut.AddDate(0, 0, 14).Format("2006-01-02"),
		}).
		Get(fmt.Sprintf("/cottage/%s/available-dates", cottageName))
	if err != nil {
		return false, err
	}
	if resp.IsError() {
		return false, fmt.Errorf("error: %s", resp.Status())
	}

	var availablePeriodDTO domain.AvailablePeriodDTO
	if err := json.Unmarshal(resp.Body(), &availablePeriodDTO); err != nil {
		return false, err
	}
	availablePeriods := availablePeriodDTO.ToPeriods()

	for _, period := range availablePeriods {
		if !period.Start.After(checkIn) && !period.End.Before(checkOut) {
			return true, nil
		}
	}

	return false, nil
}
