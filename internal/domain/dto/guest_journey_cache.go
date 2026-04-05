package dto

import (
	"github.com/Kenji-Uema/guestSimulator/internal/domain/dto/booking"
	"github.com/Kenji-Uema/guestSimulator/internal/domain/dto/guest_registration"
)

type GuestJourneyCacheValue struct {
	GuestID      string                    `json:"guestId"`
	PersonalInfo *guest_registration.Guest `json:"personalInfo,omitempty"`
	Booking      *GuestJourneyBooking      `json:"booking,omitempty"`
	Invoice      *GuestJourneyInvoice      `json:"invoice,omitempty"`
}

type GuestJourneyBooking struct {
	BookingID       string          `json:"bookingId"`
	SelectedCottage string          `json:"selectedCottage"`
	SelectedPeriod  *booking.Period `json:"selectedPeriod,omitempty"`
}

type GuestJourneyInvoice struct {
	InvoiceNumber string `json:"invoiceNumber"`
	ReceiptNumber string `json:"receiptNumber,omitempty"`
}
