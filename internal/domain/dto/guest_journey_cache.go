package dto

import "github.com/Kenji-Uema/guestSimulator/internal/domain"

type GuestJourneyCacheValue struct {
	GuestID      string               `json:"guestId"`
	PersonalInfo *domain.Guest        `json:"personalInfo,omitempty"`
	Booking      *GuestJourneyBooking `json:"booking,omitempty"`
	Invoice      *GuestJourneyInvoice `json:"invoice,omitempty"`
}

type GuestJourneyBooking struct {
	BookingID       string         `json:"bookingId"`
	SelectedCottage string         `json:"selectedCottage"`
	SelectedPeriod  *domain.Period `json:"selectedPeriod,omitempty"`
}

type GuestJourneyInvoice struct {
	InvoiceNumber string `json:"invoiceNumber"`
	ReceiptNumber string `json:"receiptNumber,omitempty"`
}
