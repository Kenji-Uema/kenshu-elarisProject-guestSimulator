package domain

type BookingRequest struct {
	GuestId        string `json:"mainGuest"`
	NumberOfGuests int    `json:"numberOfGuests"`
	CheckInDate    string `json:"checkInDate"`
	CheckOutDate   string `json:"checkOutDate"`
}

type BookingConfirmation struct {
	Id string `json:"bookingId"`
}
