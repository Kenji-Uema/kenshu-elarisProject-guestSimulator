package config

import (
	"context"
	"log/slog"

	"github.com/caarlos0/env/v11"
)

type Secret string

func (s Secret) String() string {
	return "REDACTED"
}

type Configs struct {
	AppConfig
	ProbeConfig
	RabbitMqConnConfig
	RabbitMqConsumersConfig
	RedisConfig
	ServicesConfig
	FlowsConfig
}

type AppConfig struct {
	Name      NameConfig
	Telemetry TelemetryConfig
}

type NameConfig struct {
	ServiceName string `env:"SERVICE_NAME"`
	Version     string `env:"VERSION"`
}

type ProbeConfig struct {
	Address                    string `env:"PROBE_HTTP_ADDRESS,required"`
	Port                       int    `env:"PROBE_HTTP_PORT,required"`
	ReadHeaderTimeoutInSeconds int    `env:"READ_HEADER_TIMEOUT_IN_SECONDS,required" envDefault:"5"`
	ReadTimeoutInSeconds       int    `env:"READ_TIMEOUT_IN_SECONDS,required" envDefault:"10"`
	WriteTimeoutInSeconds      int    `env:"WRITE_TIMEOUT_IN_SECONDS,required" envDefault:"15"`
	IdleTimeoutInSeconds       int    `env:"IDLE_TIMEOUT_IN_SECONDS,required" envDefault:"60"`
}

type RedisConfig struct {
	Username Secret `env:"REDIS_USERNAME" envDefault:""`
	Password Secret `env:"REDIS_PASSWORD" envDefault:""`
	Host     string `env:"REDIS_HOST,required" envDefault:"localhost"`
	Port     int    `env:"REDIS_PORT,required" envDefault:"6379"`
	DB       int    `env:"REDIS_DB" envDefault:"0"`
}

type RabbitMqConsumersConfig struct {
	HourChange    RabbitMqConsumerConfig `envPrefix:"HOUR_CHANGE_"`
	Communication RabbitMqConsumerConfig `envPrefix:"COMMUNICATION_"`
}

type RabbitMqConnConfig struct {
	Username Secret `env:"RABBITMQ_USERNAME,required"`
	Password Secret `env:"RABBITMQ_PASSWORD,required"`
	Host     string `env:"RABBITMQ_HOST,required"`
	Port     int    `env:"RABBITMQ_PORT,required"`
}

type ServicesConfig struct {
	ClockEmuGrpcUrl      string `env:"CLOCK_EMU_GRPC_URL,required"`
	ClockEmuGrpcPort     int    `env:"CLOCK_EMU_GRPC_PORT,required"`
	CottageManagerUrl    string `env:"COTTAGE_MANAGER_URL,required"`
	CottageManagerPort   int    `env:"COTTAGE_MANAGER_PORT,required"`
	GuestManagerUrl      string `env:"GUEST_MANAGER_URL,required"`
	GuestManagerPort     int    `env:"GUEST_MANAGER_PORT,required"`
	PaymentSimulatorUrl  string `env:"PAYMENT_SIMULATOR_URL,required"`
	PaymentSimulatorPort int    `env:"PAYMENT_SIMULATOR_PORT,required"`
}

type FlowsConfig struct {
	Booking       BookingFlowConfig
	Payment       PaymentFlowConfig
	Lodging       LodgingFlowConfig
	GuestRegister GuestRegisterFlowConfig
	GuestJourney  GuestJourneyFlowConfig
}

type BookingFlowConfig struct {
	GraphFile        string `env:"BOOKING_FLOW_GRAPH_FILE" envDefault:"docs/booking_mdp.dot"`
	ConcurrencyLevel int    `env:"BOOKING_FLOW_CONCURRENCY_LEVEL,required"`
}

type LodgingFlowConfig struct {
	ConcurrencyLevel int `env:"LODGING_FLOW_CONCURRENCY_LEVEL,required"`
}

type PaymentFlowConfig struct {
	ConcurrencyLevel int `env:"PAYMENT_FLOW_CONCURRENCY_LEVEL,required"`
}

type GuestRegisterFlowConfig struct {
	GraphFile        string `env:"GUEST_REGISTER_GRAPH_FILE" envDefault:"docs/guest_register_mdp.dot"`
	ConcurrencyLevel int    `env:"GUEST_REGISTER_FLOW_CONCURRENCY_LEVEL,required"`
}

type GuestJourneyFlowConfig struct {
	ConcurrencyLevel int `env:"GUEST_JOURNEY_FLOW_CONCURRENCY_LEVEL,required"`
}

type TelemetryConfig struct {
	OTLPEndpoint   string `env:"OTEL_EXPORTER_OTLP_ENDPOINT,required"`
	OTLPGrpcPort   int    `env:"OTEL_EXPORTER_OTLP_GRPC_PORT,required"`
	OTLPHealthPort int    `env:"OTEL_EXPORTER_OTLP_HEALTH_PORT,required"`
	OTLPInsecure   bool   `env:"OTEL_EXPORTER_OTLP_INSECURE,required"`
}

func LoadConfigs() (Configs, error) {
	var cfg Configs
	if err := env.Parse(&cfg); err != nil {
		return cfg, err
	}

	slog.InfoContext(context.Background(), "config loaded", "config", cfg)
	return cfg, nil
}
