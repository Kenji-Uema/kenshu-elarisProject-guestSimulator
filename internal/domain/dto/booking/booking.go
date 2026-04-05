package booking

type Request struct {
	GuestId        string `json:"mainGuest"`
	NumberOfGuests int    `json:"numberOfGuests"`
	CheckInDate    string `json:"checkInDate"`
	CheckOutDate   string `json:"checkOutDate"`
	GuestName      string `json:"guestName"`
	GuestEmail     string `json:"guestEmail"`
	GuestDocument  string `json:"guestDocument"`
	BillingAddress string `json:"billingAddress"`
}

type Confirmation struct {
	Id string `json:"bookingId"`
}
