package register_guest_state

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Kenji-Uema/guestEmulator/internal/domain"
	"github.com/go-resty/resty/v2"
)

type RetrieveGuestState struct {
	client *resty.Client
}

func NewRetrieveGuestState(c *resty.Client) *RetrieveGuestState {
	return &RetrieveGuestState{client: c}
}

func (s RetrieveGuestState) Execute(ctx context.Context, guestId string) (domain.IgnoredField, error) {
	slog.Info("User retrieves its own account")

	resp, err := s.client.R().
		SetContext(ctx).
		Get(fmt.Sprintf("/guest/%s", guestId))

	if err != nil {
		return domain.IgnoredField{}, err
	}

	if resp.IsError() {
		return domain.IgnoredField{}, fmt.Errorf("error: %s", resp.Status())
	}

	slog.Info(string(resp.Body()))

	return domain.IgnoredField{}, nil
}
