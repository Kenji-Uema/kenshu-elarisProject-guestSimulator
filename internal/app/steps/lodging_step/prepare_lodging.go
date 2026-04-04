package lodging_step

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/Kenji-Uema/guestSimulator/internal/app/journeyctx"
	"github.com/Kenji-Uema/guestSimulator/internal/app/steps"
	"github.com/Kenji-Uema/guestSimulator/internal/domain"
	"github.com/Kenji-Uema/guestSimulator/internal/domain/dto"
	redisc "github.com/Kenji-Uema/guestSimulator/internal/infra/redis"
	"github.com/Kenji-Uema/guestSimulator/internal/infra/telemetry"
	"github.com/Kenji-Uema/guestSimulator/internal/transport/grpc"
	"github.com/go-resty/resty/v2"
)

type PrepareLodgingStep struct {
	clock         *grpc.Emu
	cottageClient *resty.Client
	state         *domain.State
	redis         *redisc.Redis
}

func NewPrepareLodgingStep(state *domain.State, clock *grpc.Emu, cottageClient *resty.Client, redis *redisc.Redis) steps.Step {
	return &PrepareLodgingStep{clock: clock, cottageClient: cottageClient, state: state, redis: redis}
}

func (s PrepareLodgingStep) Name() string {
	return "PrepareLodgingStep"
}

func (s PrepareLodgingStep) Validate() error {
	if s.state == nil {
		return fmt.Errorf("invalid state, state is nil")
	}
	if s.redis == nil {
		return fmt.Errorf("invalid redis client")
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

		cacheValue, err := journeyctx.Load(ctx, s.redis, s.state)
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
		if err := journeyctx.Save(ctx, s.redis, s.state, cacheValue); err != nil {
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
