package booking_step

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/Kenji-Uema/guestEmulator/internal/app/steps"
	"github.com/Kenji-Uema/guestEmulator/internal/domain"
	"github.com/Kenji-Uema/guestEmulator/internal/tooling/telemetry"

	"github.com/go-resty/resty/v2"
)

type ListCottagesStep struct {
	client *resty.Client
	state  *domain.State
}

func NewListCottagesStep(state *domain.State, c *resty.Client) steps.Step {
	return &ListCottagesStep{client: c, state: state}
}

func (s ListCottagesStep) Name() string {
	return "ListCottagesStep"
}

func (s ListCottagesStep) Validate() error {
	if s.state == nil {
		return fmt.Errorf("invalid state, state is nil")
	}
	return nil
}

func (s ListCottagesStep) Execute(ctx context.Context) error {
	ctx, span := telemetry.Tracer.Start(ctx, "ListCottagesStep")
	defer span.End()

	slog.InfoContext(ctx, "User retrieves the list of cottages")

	resp, err := s.client.R().
		SetContext(ctx).
		Get("/cottages")

	if err != nil {
		return err
	}

	if resp.IsError() {
		return fmt.Errorf("error: %s", resp.Status())
	}

	var cottages []domain.Cottage
	if err := json.Unmarshal(resp.Body(), &cottages); err != nil {
		return err
	}

	cottageNames := make([]string, len(cottages))
	for i, c := range cottages {
		cottageNames[i] = c.Name
	}

	s.state.CottageNames = cottageNames
	slog.InfoContext(ctx, "state updated, added list of cottage names", "cottagesNames", cottageNames)

	return nil
}
