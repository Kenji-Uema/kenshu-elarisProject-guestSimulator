package domain

type Sender string

const (
	SenderGuest  Sender = "SENDER_GUEST"
	SenderSystem Sender = "SENDER_SYSTEM"
)

type GuestAction string

const (
	GuestActionShowForCheckin     GuestAction = "SHOW_FOR_CHECKIN"
	GuestActionEnterCottage       GuestAction = "ENTER_COTTAGE"
	GuestActionGoForABath         GuestAction = "GO_FOR_A_BATH"
	GuestActionGoForDinner        GuestAction = "GO_FOR_DINNER"
	GuestActionGoToSleep          GuestAction = "GO_TO_SLEEP"
	GuestActionLeaveCottage       GuestAction = "LEAVE_COTTAGE"
	GuestActionProceedToCheckout  GuestAction = "PROCEED_TO_CHECKOUT"
	GuestActionReturnCottageKey   GuestAction = "RETURN_COTTAGE_KEY"
	GuestActionTakeCottageKey     GuestAction = "TAKE_COTTAGE_KEY"
	GuestActionWakeup             GuestAction = "WAKEUP"
	GuestActionGoForBreakfast     GuestAction = "GO_FOR_BREAKFAST"
	GuestActionLeaveCleanupNotice GuestAction = "LEAVE_CLEANUP_NOTIFICATION"
	GuestActionEnjoyResort        GuestAction = "ENJOY_RESORT"
)

type SystemNotification string

const (
	SystemNotificationBookingChecking  SystemNotification = "BOOKING_CHECKING"
	SystemNotificationCheckInComplete  SystemNotification = "CHECK_IN_COMPLETE"
	SystemNotificationDinnerReady      SystemNotification = "DINNER_READY"
	SystemNotificationBreakfastReady   SystemNotification = "BREAKFAST_READY"
	SystemNotificationCheckOutToday    SystemNotification = "CHECK_OUT_TODAY"
	SystemNotificationCheckOutComplete SystemNotification = "CHECK_OUT_COMPLETE"
)

type SystemRequest string

const (
	SystemRequestDocument       SystemRequest = "REQUEST_DOCUMENT"
	SystemRequestBookingNumber  SystemRequest = "REQUEST_BOOKING_NUMBER"
	SystemRequestGiveCottageKey SystemRequest = "GIVE_COTTAGE_KEY"
	SystemRequestCottageKey     SystemRequest = "REQUEST_COTTAGE_KEY"
)

type ChatMessage struct {
	MessageID          string             `json:"messageId,omitempty"`
	CorrelationID      string             `json:"correlationId,omitempty"`
	Sender             Sender             `json:"sender,omitempty"`
	ProtocolVersion    string             `json:"protocolVersion,omitempty"`
	TraceContext       map[string]string  `json:"traceContext,omitempty"`
	GuestAction        GuestAction        `json:"guestAction,omitempty"`
	GuestResponse      *GuestResponse     `json:"guestResponse,omitempty"`
	SystemNotification SystemNotification `json:"systemNotification,omitempty"`
	SystemRequest      SystemRequest      `json:"systemRequest,omitempty"`
	Ack                *Ack               `json:"ack,omitempty"`
}

type GuestResponse struct {
	ShowDocument      *ShowDocument      `json:"showDocument,omitempty"`
	ShowBookingNumber *ShowBookingNumber `json:"showBookingNumber,omitempty"`
	ReceiveCottageKey *ReceiveCottageKey `json:"receiveCottageKey,omitempty"`
	ReturnCottageKey  *ReturnCottageKey  `json:"returnCottageKey,omitempty"`
}

type ShowDocument struct {
	DocumentID string `json:"documentId"`
}

type ShowBookingNumber struct {
	BookingID string `json:"bookingId"`
}

type ReceiveCottageKey struct {
	CottageKeyID string `json:"cottageKeyId"`
}

type ReturnCottageKey struct {
	CottageKeyID string `json:"cottageKeyId"`
}

type Ack struct {
	AcknowledgedMessageID string `json:"acknowledgedMessageId"`
	Status                string `json:"status"`
	Code                  string `json:"code"`
}
