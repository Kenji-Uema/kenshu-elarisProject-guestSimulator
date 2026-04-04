package domain

type State struct {
	Guest        *Guest
	GuestId      string
	CottageNames []string
	RedisKey     string
	QueueName    string
	RoutingKey   string
}
