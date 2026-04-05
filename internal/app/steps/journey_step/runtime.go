package journey_step

import (
	"sync"

	"github.com/Kenji-Uema/guestSimulator/internal/port"
	amqp "github.com/rabbitmq/amqp091-go"
)

type GuestCommunicationRuntime struct {
	Consumer   port.RabbitConsumer
	Deliveries <-chan amqp.Delivery
	Pending    []amqp.Delivery
	mu         sync.Mutex
}
