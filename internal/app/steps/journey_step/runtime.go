package journey_step

import (
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/Kenji-Uema/guestSimulator/internal/infra/mq"
)

type GuestCommunicationRuntime struct {
	Consumer   *mq.RabbitmqConsumer
	Deliveries <-chan amqp.Delivery
	Pending    []amqp.Delivery
	mu         sync.Mutex
}
