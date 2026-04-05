package domain

import "github.com/Kenji-Uema/guestSimulator/internal/domain/dto/guest_registration"

type State struct {
	Guest        *guest_registration.Guest
	GuestId      string
	CottageNames []string
	RedisKey     string
	QueueName    string
	RoutingKey   string
}
