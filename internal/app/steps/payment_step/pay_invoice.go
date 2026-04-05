package payment_step

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/Kenji-Uema/guestSimulator/internal/app/steps"
	"github.com/Kenji-Uema/guestSimulator/internal/domain"
	"github.com/Kenji-Uema/guestSimulator/internal/domain/dto"
	"github.com/Kenji-Uema/guestSimulator/internal/infra/telemetry"
	"github.com/Kenji-Uema/guestSimulator/internal/port"
	"github.com/go-resty/resty/v2"
)

type PayInvoiceStep struct {
	paymentClient *resty.Client
	cache         port.Cache
	state         *domain.State
}

func NewPayInvoiceStep(state *domain.State, paymentClient *resty.Client, cache port.Cache) steps.Step {
	return &PayInvoiceStep{
		paymentClient: paymentClient,
		cache:         cache,
		state:         state,
	}
}

func (s PayInvoiceStep) Name() string { return "PayInvoiceStep" }

func (s PayInvoiceStep) Validate() error {
	if s.state == nil {
		return fmt.Errorf("invalid state, state is nil")
	}
	if s.state.GuestId == "" {
		return fmt.Errorf("invalid state, guestId is empty")
	}
	if s.cache == nil {
		return fmt.Errorf("invalid guest journey cache")
	}
	if s.paymentClient == nil {
		return fmt.Errorf("invalid payment client")
	}
	return nil
}

func (s PayInvoiceStep) Execute(ctx context.Context) error {
	ctx, span := telemetry.Tracer.Start(ctx, "PayInvoiceStep")
	defer span.End()

	if err := s.Validate(); err != nil {
		return err
	}

	cacheValue, err := s.cache.Load(ctx, s.state)
	if err != nil {
		return err
	}
	if cacheValue.PersonalInfo == nil {
		return fmt.Errorf("invalid cached guest context")
	}
	if cacheValue.Booking == nil || cacheValue.Booking.BookingID == "" || cacheValue.Booking.SelectedCottage == "" || cacheValue.Booking.SelectedPeriod == nil {
		return fmt.Errorf("invalid cached booking context")
	}
	if cacheValue.Invoice == nil || cacheValue.Invoice.InvoiceNumber == "" {
		return fmt.Errorf("invalid cached invoice context")
	}

	holderName := strings.TrimSpace(cacheValue.PersonalInfo.GivenNames + " " + cacheValue.PersonalInfo.Surname)

	resp, err := s.paymentClient.R().
		SetContext(ctx).
		SetBody(dto.PayWithCardRequest{
			Number:     "4111111111111111",
			Brand:      "VISA",
			ExpMonth:   12,
			ExpYear:    2030,
			Cvv:        "123",
			HolderName: holderName,
		}).
		Post(fmt.Sprintf("/v1/payments/invoice/%s", cacheValue.Invoice.InvoiceNumber))
	if err != nil {
		return err
	}
	if resp.IsError() {
		return fmt.Errorf("pay invoice %s: %s", cacheValue.Invoice.InvoiceNumber, resp.Status())
	}

	var paymentResp dto.PayWithCardResponse
	if err := json.Unmarshal(resp.Body(), &paymentResp); err != nil {
		return err
	}
	if paymentResp.Status != "PAYMENT_STATUS_SUCCEEDED" {
		return fmt.Errorf("payment for invoice %s returned status %s", cacheValue.Invoice.InvoiceNumber, paymentResp.Status)
	}

	cacheValue.Invoice.ReceiptNumber = paymentResp.ReceiptNumber
	if err := s.cache.Save(ctx, s.state, cacheValue); err != nil {
		return err
	}
	slog.InfoContext(ctx, "invoice paid",
		"bookingId", cacheValue.Booking.BookingID,
		"invoiceNumber", cacheValue.Invoice.InvoiceNumber,
		"receiptNumber", paymentResp.ReceiptNumber)

	return nil
}
