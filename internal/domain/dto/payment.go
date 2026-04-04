package dto

import "time"

type ReissuePaymentRequest struct {
	BookingNumber  string `json:"bookingNumber"`
	DocumentNumber string `json:"documentNumber"`
}

type PaymentRequestResponse struct {
	InvoiceNumber string `json:"invoiceNumber"`
}

type PayWithCardRequest struct {
	Number     string `json:"number"`
	Brand      string `json:"brand"`
	ExpMonth   uint32 `json:"expMonth"`
	ExpYear    uint32 `json:"expYear"`
	Cvv        string `json:"cvv"`
	HolderName string `json:"holderName"`
}

type PayWithCardResponse struct {
	ReceiptNumber string `json:"receiptNumber"`
	InvoiceNumber string `json:"invoiceNumber"`
	Status        string `json:"status"`
}

type GuestBooking struct {
	NumberOfGuests int                `json:"number_of_guests"`
	StayPeriod     GuestBookingPeriod `json:"stay_period"`
	CottageName    string             `json:"cottage_name"`
	Status         string             `json:"status"`
}

type GuestBookingPeriod struct {
	CheckIn  time.Time `json:"checkin"`
	CheckOut time.Time `json:"checkout"`
}
