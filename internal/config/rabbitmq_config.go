package config

type RabbitMqConfig struct {
	Conn      RabbitMqConnConfig
	TimeEvent TimeEventConsumerConfig
}

type RabbitMqConnConfig struct {
	Username Secret `env:"RABBITMQ_USERNAME,required"`
	Password Secret `env:"RABBITMQ_PASSWORD,required"`
	Host     string `env:"RABBITMQ_HOST,required"`
	Port     int    `env:"RABBITMQ_PORT,required"`
}

type RabbitMqProducerConfig struct {
	Exchange ExchangeConfig `envPrefix:"EXCHANGE_"`
	Publish  PublishConfig  `envPrefix:"PUBLISH_"`
}

type RabbitMqConsumerConfig struct {
	Queue   QueueConfig   `envPrefix:"QUEUE_"`
	Binding BindingConfig `envPrefix:"BINDING_"`
	Consume ConsumeConfig `envPrefix:"CONSUME_"`
}

type TimeEventConsumerConfig struct {
	QueueName    string `env:"TIME_EVENT_HOUR_QUEUE_NAME,required"`
	ExchangeName string `env:"TIME_EVENT_EXCHANGE_NAME,required"`
	RoutingKey   string `env:"TIME_EVENT_HOUR_ROUTING_KEY" envDefault:""`
	NoWait       bool   `env:"TIME_EVENT_NO_WAIT" envDefault:"false"`
	Consumer     string `env:"TIME_EVENT_CONSUMER" envDefault:""`
	AutoAck      bool   `env:"TIME_EVENT_AUTO_ACK" envDefault:"false"`
	Exclusive    bool   `env:"TIME_EVENT_EXCLUSIVE" envDefault:"false"`
	NoLocal      bool   `env:"TIME_EVENT_NO_LOCAL" envDefault:"false"`
}

func (c TimeEventConsumerConfig) Queue() QueueConfig {
	return QueueConfig{
		Name:       c.QueueName,
		Durable:    true,
		AutoDelete: false,
		Exclusive:  false,
		NoWait:     c.NoWait,
	}
}

func (c TimeEventConsumerConfig) Binding() BindingConfig {
	return BindingConfig{
		ExchangeName: c.ExchangeName,
		RoutingKey:   c.RoutingKey,
		NoWait:       c.NoWait,
	}
}

func (c TimeEventConsumerConfig) Consume() ConsumeConfig {
	return ConsumeConfig{
		Consumer:  c.Consumer,
		AutoAck:   c.AutoAck,
		Exclusive: c.Exclusive,
		NoLocal:   c.NoLocal,
		NoWait:    c.NoWait,
	}
}

type ExchangeConfig struct {
	Name       string `env:"NAME,required" envDefault:"ex.communication"`
	Kind       string `env:"KIND,required" envDefault:"direct"`
	Durable    bool   `env:"DURABLE" envDefault:"true"`
	AutoDelete bool   `env:"AUTO_DELETE" envDefault:"false"`
	Internal   bool   `env:"INTERNAL" envDefault:"false"`
	NoWait     bool   `env:"NO_WAIT" envDefault:"false"`
}

type QueueConfig struct {
	Name       string `env:"NAME,required"`
	Durable    bool   `env:"DURABLE" envDefault:"true"`
	AutoDelete bool   `env:"AUTO_DELETE" envDefault:"false"`
	Exclusive  bool   `env:"EXCLUSIVE" envDefault:"false"`
	NoWait     bool   `env:"NO_WAIT" envDefault:"false"`
}

type BindingConfig struct {
	ExchangeName string `env:"EXCHANGE_NAME,required"`
	RoutingKey   string `env:"ROUTING_KEY,required"`
	NoWait       bool   `env:"NO_WAIT" envDefault:"false"`
}

type PublishConfig struct {
	Mandatory bool `env:"MANDATORY" envDefault:"false"`
	Immediate bool `env:"IMMEDIATE" envDefault:"false"`
}

type ConsumeConfig struct {
	Consumer  string `env:"CONSUMER"`
	AutoAck   bool   `env:"AUTO_ACK" envDefault:"false"`
	Exclusive bool   `env:"EXCLUSIVE" envDefault:"false"`
	NoLocal   bool   `env:"NO_LOCAL" envDefault:"false"`
	NoWait    bool   `env:"NO_WAIT" envDefault:"false"`
}
