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
	RabbitMqConfig
	RedisConfig
	ServicesConfig
	MachinesConfig
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

type MachinesConfig struct {
	Booking       BookingMachineConfig
	Payment       PaymentMachineConfig
	Lodging       LodgingMachineConfig
	GuestRegister GuestRegisterMachineConfig
	GuestJourney  GuestJourneyMachineConfig
}

type BookingMachineConfig struct {
	GraphFile                 string `env:"BOOKING_MACHINE_GRAPH_FILE" envDefault:"docs/booking_mdp.dot"`
	ConcurrencyLevel          int    `env:"BOOKING_MACHINE_CONCURRENCY_LEVEL,required"`
	TimeBetweenStepsInSeconds int    `env:"BOOKING_MACHINE_TIME_BETWEEN_STEPS_IN_SECONDS,required"`
}

type LodgingMachineConfig struct {
	ConcurrencyLevel          int `env:"LODGING_MACHINE_CONCURRENCY_LEVEL,required"`
	TimeBetweenStepsInSeconds int `env:"LODGING_MACHINE_TIME_BETWEEN_STEPS_IN_SECONDS,required"`
}

type PaymentMachineConfig struct {
	ConcurrencyLevel          int `env:"PAYMENT_MACHINE_CONCURRENCY_LEVEL,required"`
	TimeBetweenStepsInSeconds int `env:"PAYMENT_MACHINE_TIME_BETWEEN_STEPS_IN_SECONDS,required"`
}

type GuestRegisterMachineConfig struct {
	GraphFile                 string `env:"GUEST_REGISTER_GRAPH_FILE" envDefault:"docs/guest_register_mdp.dot"`
	ConcurrencyLevel          int    `env:"GUEST_REGISTER_MACHINE_CONCURRENCY_LEVEL,required"`
	TimeBetweenStepsInSeconds int    `env:"GUEST_REGISTER_TIME_BETWEEN_STEPS_IN_SECONDS,required"`
}

type GuestJourneyMachineConfig struct {
	ConcurrencyLevel          int `env:"GUEST_JOURNEY_MACHINE_CONCURRENCY_LEVEL,required"`
	TimeBetweenStepsInSeconds int `env:"GUEST_JOURNEY_TIME_BETWEEN_STEPS_IN_SECONDS,required"`
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
