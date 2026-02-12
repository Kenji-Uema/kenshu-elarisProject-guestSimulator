package booking_state

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/Kenji-Uema/guestEmulator/internal/domain"

	"github.com/go-resty/resty/v2"
)

type ListCottagesState struct {
	client *resty.Client
}

func NewListCottagesState(c *resty.Client) *ListCottagesState {
	return &ListCottagesState{client: c}
}

func (s *ListCottagesState) Execute(ctx context.Context, _ domain.IgnoredField) ([]string, error) {
	slog.Info("User retrieves the list of cottages")

	resp, err := s.client.R().
		SetContext(ctx).
		Get("/cottages")

	if err != nil {
		return nil, err
	}

	if resp.IsError() {
		return nil, fmt.Errorf("error: %s", resp.Status())
	}

	var cottages []domain.Cottage
	if err := json.Unmarshal(resp.Body(), &cottages); err != nil {
		return nil, err
	}

	cottageNames := make([]string, len(cottages))
	for _, c := range cottages {
		cottageNames = append(cottageNames, c.Name)
	}

	return cottageNames, nil
}
