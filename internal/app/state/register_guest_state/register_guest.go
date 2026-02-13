package register_guest_state

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/Kenji-Uema/guestEmulator/internal/domain"
	"github.com/go-resty/resty/v2"
)

type RegisterGuestState struct {
	client *resty.Client
}

func NewRegisterGuestState(c *resty.Client) *RegisterGuestState {
	return &RegisterGuestState{client: c}
}

func (s RegisterGuestState) Execute(ctx context.Context, _ domain.IgnoredField) (string, error) {
	slog.Info("User registers its own account")

	guest := ctx.Value("guest").(domain.Guest)

	resp, err := s.client.R().
		SetContext(ctx).
		SetBody(guest).
		Post("/guest")

	if err != nil {
		return "", err
	}

	if resp.IsError() {
		return "", fmt.Errorf("error: %s", resp.Error())
	}

	var guestId string
	if err := json.Unmarshal(resp.Body(), &guestId); err != nil {
		return "", err
	}
	return guestId, nil

}
