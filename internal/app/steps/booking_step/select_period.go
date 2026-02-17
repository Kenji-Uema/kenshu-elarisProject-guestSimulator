package booking_step

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/Kenji-Uema/guestEmulator/internal/app/steps"
	"github.com/Kenji-Uema/guestEmulator/internal/app/utils"
	"github.com/Kenji-Uema/guestEmulator/internal/domain"
	"github.com/Kenji-Uema/guestEmulator/internal/tooling/telemetry"
	"github.com/Kenji-Uema/guestEmulator/internal/transport/grpc"
	"github.com/go-resty/resty/v2"
)

var numberOfNights = []int{3, 5, 7, 10, 14}
var daysAhead = []int{5, 7, 14, 30, 45, 60, 90, 120}
var window = 30

type SelectPeriodStep struct {
	clock  *grpc.Emu
	client *resty.Client
	state  *domain.State
}

func NewSelectPeriodStep(state *domain.State, clock *grpc.Emu, client *resty.Client) steps.Step {
	return &SelectPeriodStep{clock: clock, client: client, state: state}
}

func (s SelectPeriodStep) Name() string {
	return "SelectPeriodStep"
}

func (s SelectPeriodStep) Validate() error {
	if s.state == nil {
		return fmt.Errorf("invalid state, state is nil")
	}

	if s.state.SelectedCottage == "" {
		return fmt.Errorf("invalid state, selectedCottage is empty")
	}

	return nil
}

func (s SelectPeriodStep) Execute(ctx context.Context) error {
	ctx, span := telemetry.Tracer.Start(ctx, "SelectPeriodStep")
	defer span.End()

	slog.InfoContext(ctx, "User selects a period of time")

	nights := utils.PickRandom(numberOfNights)
	searchPeriod := utils.PickRandom(daysAhead)

	now, err := s.clock.Now(ctx)
	if err != nil {
		return err
	}
	from := now.AddDate(0, 0, searchPeriod)
	to := from.AddDate(0, 0, searchPeriod+window)

	resp, err := s.client.R().
		SetContext(ctx).
		SetQueryParams(map[string]string{"to": to.Format("2006-01-02"), "from": from.Format("2006-01-02")}).
		Get(fmt.Sprintf("/cottage/%s/available-dates", s.state.SelectedCottage))

	if err != nil {
		return err
	}

	if resp.IsError() {
		return fmt.Errorf("error: %s", resp.Status())
	}

	var availablePeriods []domain.Period
	if err := json.Unmarshal(resp.Body(), &availablePeriods); err != nil {
		return err
	}

	for _, period := range availablePeriods {
		if period.End.Sub(period.Start).Hours()-float64(24*nights) >= 0 {
			s.state.SelectedPeriod = &period
			slog.InfoContext(ctx, "state updated, stay period selected", "selectedPeriod", period)
			return nil
		}
	}

	slog.WarnContext(ctx, "No suitable period found")
	return nil
}
