package config

type RabbitMqConsumerConfig struct {
	Queue   QueueConfig   `envPrefix:"QUEUE_"`
	Binding BindingConfig `envPrefix:"BINDING_"`
	Consume ConsumeConfig `envPrefix:"CONSUME_"`
}

type QueueConfig struct {
	Name       string `env:"NAME"`
	Durable    bool   `env:"DURABLE" envDefault:"true"`
	AutoDelete bool   `env:"AUTO_DELETE" envDefault:"false"`
	Exclusive  bool   `env:"EXCLUSIVE" envDefault:"false"`
	NoWait     bool   `env:"NO_WAIT" envDefault:"false"`
}

type BindingConfig struct {
	ExchangeName string `env:"EXCHANGE_NAME"`
	RoutingKey   string `env:"ROUTING_KEY" envDefault:""`
	NoWait       bool   `env:"NO_WAIT" envDefault:"false"`
}

type ConsumeConfig struct {
	Consumer  string `env:"CONSUMER"`
	AutoAck   bool   `env:"AUTO_ACK" envDefault:"false"`
	Exclusive bool   `env:"EXCLUSIVE" envDefault:"false"`
	NoLocal   bool   `env:"NO_LOCAL" envDefault:"false"`
	NoWait    bool   `env:"NO_WAIT" envDefault:"false"`
}
