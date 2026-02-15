package booking_state

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/Kenji-Uema/guestEmulator/internal/domain"
	"github.com/Kenji-Uema/guestEmulator/internal/tooling/telemetry"

	"github.com/go-resty/resty/v2"
)

type SearchByTypeAndPeriodState struct {
	client *resty.Client
}

func NewSearchByTypeAndPeriodState(c *resty.Client) *SearchByTypeAndPeriodState {
	return &SearchByTypeAndPeriodState{client: c}
}

func (s SearchByTypeAndPeriodState) Execute(ctx context.Context, _ domain.IgnoredField) ([]domain.CottageAvailable, error) {
	ctx, span := telemetry.Tracer.Start(ctx, "SearchByTypeAndPeriodState")
	defer span.End()

	slog.InfoContext(ctx, "User search for cottages by type and period")

	resp, err := s.client.R().
		SetContext(ctx).
		Get("/cottages")

	if err != nil {
		return nil, err
	}

	if resp.IsError() {
		return nil, fmt.Errorf("error: %s", resp.Status())
	}

	var cottages []domain.CottageAvailable
	if err := json.Unmarshal(resp.Body(), &cottages); err != nil {
		return nil, err
	}

	return cottages, nil
}
