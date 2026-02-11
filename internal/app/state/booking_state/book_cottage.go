package booking_state

import (
	"context"
	"encoding/json"
	"fmt"
	"guestEmulator/internal/domain"
	"log"

	"github.com/go-resty/resty/v2"
)

type BookCottageState struct {
	client *resty.Client
}

func NewBookCottageState(c *resty.Client) *BookCottageState {
	return &BookCottageState{client: c}
}

func (b BookCottageState) Execute(ctx context.Context, in domain.Cottage) (domain.BookingConfirmation, error) {
	log.Printf("User book a cottage %v\n", in)

	resp, err := b.client.R().
		SetContext(ctx).
		Get("/cottages")

	if err != nil {
		return domain.BookingConfirmation{}, err
	}

	if resp.IsError() {
		return domain.BookingConfirmation{}, fmt.Errorf("error: %s", resp.Status())
	}

	var bookingConfirmation domain.BookingConfirmation
	if err := json.Unmarshal(resp.Body(), &bookingConfirmation); err != nil {
		return domain.BookingConfirmation{}, err
	}

	return bookingConfirmation, nil
}
