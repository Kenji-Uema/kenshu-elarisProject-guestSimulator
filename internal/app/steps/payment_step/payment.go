package payment_step

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/Kenji-Uema/guestSimulator/internal/app/journeyctx"
	"github.com/Kenji-Uema/guestSimulator/internal/app/steps"
	"github.com/Kenji-Uema/guestSimulator/internal/domain"
	"github.com/Kenji-Uema/guestSimulator/internal/domain/dto"
	redisc "github.com/Kenji-Uema/guestSimulator/internal/infra/redis"
	"github.com/Kenji-Uema/guestSimulator/internal/infra/telemetry"
	"github.com/go-resty/resty/v2"
)

const (
	invoicePollInterval        = 1 * time.Second
	invoicePollMaxAttempts     = 90
	bookingConfirmPollInterval = 1 * time.Second
	bookingConfirmMaxAttempts  = 90
)

type paymentRuntime struct {
	guestClient   *resty.Client
	paymentClient *resty.Client
	redis         *redisc.Redis
	state         *domain.State
}

type WaitForInvoiceStep struct {
	paymentRuntime
}

func NewWaitForInvoiceStep(state *domain.State, guestClient *resty.Client, paymentClient *resty.Client, redis *redisc.Redis) steps.Step {
	return &WaitForInvoiceStep{
		paymentRuntime: newPaymentRuntime(state, guestClient, paymentClient, redis),
	}
}

func (s WaitForInvoiceStep) Name() string { return "WaitForInvoiceStep" }

func (s WaitForInvoiceStep) Validate() error {
	return s.validateBase()
}

func (s WaitForInvoiceStep) Execute(ctx context.Context) error {
	ctx, span := telemetry.Tracer.Start(ctx, "WaitForInvoiceStep")
	defer span.End()

	cacheValue, err := s.loadCache(ctx)
	if err != nil {
		return err
	}

	for attempt := 1; attempt <= invoicePollMaxAttempts; attempt++ {
		invoiceNumber, err := s.reissuePaymentRequest(ctx, cacheValue)
		if err == nil {
			if cacheValue.Invoice == nil {
				cacheValue.Invoice = &dto.GuestJourneyInvoice{}
			}
			cacheValue.Invoice.InvoiceNumber = invoiceNumber
			if err := journeyctx.Save(ctx, s.redis, s.state, cacheValue); err != nil {
				return err
			}
			slog.InfoContext(ctx, "invoice available for booking", "bookingId", cacheValue.Booking.BookingID, "invoiceNumber", invoiceNumber)
			return nil
		}

		slog.DebugContext(ctx, "invoice not available yet", "bookingId", cacheValue.Booking.BookingID, "attempt", attempt, "error", err)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(invoicePollInterval):
		}
	}

	return fmt.Errorf("invoice not available for booking %s after %d attempts", cacheValue.Booking.BookingID, invoicePollMaxAttempts)
}

type PayInvoiceStep struct {
	paymentRuntime
}

func NewPayInvoiceStep(state *domain.State, guestClient *resty.Client, paymentClient *resty.Client, redis *redisc.Redis) steps.Step {
	return &PayInvoiceStep{
		paymentRuntime: newPaymentRuntime(state, guestClient, paymentClient, redis),
	}
}

func (s PayInvoiceStep) Name() string { return "PayInvoiceStep" }

func (s PayInvoiceStep) Validate() error {
	return s.validateBase()
}

func (s PayInvoiceStep) Execute(ctx context.Context) error {
	ctx, span := telemetry.Tracer.Start(ctx, "PayInvoiceStep")
	defer span.End()

	cacheValue, err := s.loadCache(ctx)
	if err != nil {
		return err
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
	if err := journeyctx.Save(ctx, s.redis, s.state, cacheValue); err != nil {
		return err
	}
	slog.InfoContext(ctx, "invoice paid", "bookingId", cacheValue.Booking.BookingID, "invoiceNumber", cacheValue.Invoice.InvoiceNumber, "receiptNumber", paymentResp.ReceiptNumber)
	return nil
}

type WaitForConfirmedBookingStep struct {
	paymentRuntime
}

func NewWaitForConfirmedBookingStep(state *domain.State, guestClient *resty.Client, paymentClient *resty.Client, redis *redisc.Redis) steps.Step {
	return &WaitForConfirmedBookingStep{
		paymentRuntime: newPaymentRuntime(state, guestClient, paymentClient, redis),
	}
}

func (s WaitForConfirmedBookingStep) Name() string { return "WaitForConfirmedBookingStep" }

func (s WaitForConfirmedBookingStep) Validate() error {
	return s.validateBase()
}

func (s WaitForConfirmedBookingStep) Execute(ctx context.Context) error {
	ctx, span := telemetry.Tracer.Start(ctx, "WaitForConfirmedBookingStep")
	defer span.End()

	cacheValue, err := s.loadCache(ctx)
	if err != nil {
		return err
	}

	for attempt := 1; attempt <= bookingConfirmMaxAttempts; attempt++ {
		confirmed, err := s.bookingConfirmed(ctx, cacheValue)
		if err == nil && confirmed {
			slog.InfoContext(ctx, "booking confirmed after payment", "bookingId", cacheValue.Booking.BookingID, "attempt", attempt)
			return nil
		}
		if err != nil {
			slog.DebugContext(ctx, "failed to read guest bookings while waiting for confirmation", "attempt", attempt, "error", err)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(bookingConfirmPollInterval):
		}
	}

	return fmt.Errorf("booking %s was not confirmed after %d attempts", cacheValue.Booking.BookingID, bookingConfirmMaxAttempts)
}

func newPaymentRuntime(state *domain.State, guestClient *resty.Client, paymentClient *resty.Client, redis *redisc.Redis) paymentRuntime {
	return paymentRuntime{
		guestClient:   guestClient,
		paymentClient: paymentClient,
		redis:         redis,
		state:         state,
	}
}

func (r paymentRuntime) validateBase() error {
	if r.state == nil {
		return fmt.Errorf("invalid state, state is nil")
	}
	if r.state.GuestId == "" {
		return fmt.Errorf("invalid state, guestId is empty")
	}
	if r.redis == nil {
		return fmt.Errorf("invalid redis client")
	}
	if r.guestClient == nil {
		return fmt.Errorf("invalid guest client")
	}
	if r.paymentClient == nil {
		return fmt.Errorf("invalid payment client")
	}
	return nil
}

func (r paymentRuntime) loadCache(ctx context.Context) (dto.GuestJourneyCacheValue, error) {
	if err := r.validateBase(); err != nil {
		return dto.GuestJourneyCacheValue{}, err
	}

	cacheValue, err := journeyctx.Load(ctx, r.redis, r.state)
	if err != nil {
		return dto.GuestJourneyCacheValue{}, err
	}
	if cacheValue.PersonalInfo == nil {
		return dto.GuestJourneyCacheValue{}, fmt.Errorf("invalid cached guest context")
	}
	if cacheValue.Booking == nil || cacheValue.Booking.BookingID == "" || cacheValue.Booking.SelectedCottage == "" || cacheValue.Booking.SelectedPeriod == nil {
		return dto.GuestJourneyCacheValue{}, fmt.Errorf("invalid cached booking context")
	}

	return cacheValue, nil
}

func sameUTCDay(a time.Time, b time.Time) bool {
	ay, am, ad := a.UTC().Date()
	by, bm, bd := b.UTC().Date()
	return ay == by && am == bm && ad == bd
}

func (s WaitForInvoiceStep) reissuePaymentRequest(ctx context.Context, cacheValue dto.GuestJourneyCacheValue) (string, error) {
	resp, err := s.paymentClient.R().
		SetContext(ctx).
		SetBody(dto.ReissuePaymentRequest{
			BookingNumber:  cacheValue.Booking.BookingID,
			DocumentNumber: cacheValue.PersonalInfo.DocumentId,
		}).
		Post("/v1/payments/payment_request/reissue")
	if err != nil {
		return "", err
	}
	if resp.IsError() {
		return "", fmt.Errorf("reissue payment request: %s", resp.Status())
	}

	var paymentRequest dto.PaymentRequestResponse
	if err := json.Unmarshal(resp.Body(), &paymentRequest); err != nil {
		return "", err
	}
	if paymentRequest.InvoiceNumber == "" {
		return "", fmt.Errorf("reissue payment request returned empty invoice number")
	}

	return paymentRequest.InvoiceNumber, nil
}

func (s WaitForConfirmedBookingStep) bookingConfirmed(ctx context.Context, cacheValue dto.GuestJourneyCacheValue) (bool, error) {
	resp, err := s.guestClient.R().
		SetContext(ctx).
		Get(fmt.Sprintf("/guest/%s/bookings", s.state.GuestId))
	if err != nil {
		return false, err
	}
	if resp.IsError() {
		return false, fmt.Errorf("get guest bookings: %s", resp.Status())
	}

	var bookings []dto.GuestBooking
	if err := json.Unmarshal(resp.Body(), &bookings); err != nil {
		return false, err
	}

	for _, booking := range bookings {
		if booking.CottageName != cacheValue.Booking.SelectedCottage {
			continue
		}
		if !sameUTCDay(booking.StayPeriod.CheckIn, cacheValue.Booking.SelectedPeriod.Start) {
			continue
		}
		if !sameUTCDay(booking.StayPeriod.CheckOut, cacheValue.Booking.SelectedPeriod.End) {
			continue
		}
		if strings.EqualFold(booking.Status, "confirmed") {
			return true, nil
		}
	}

	return false, nil
}
