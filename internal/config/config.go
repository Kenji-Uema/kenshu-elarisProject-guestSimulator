package config

import (
	"log/slog"

	"github.com/caarlos0/env/v11"
)

type Configs struct {
	AppConfig
	ProbeConfig
	ServicesConfig
	BookingMachineConfig
	GuestRegisterMachineConfig
	TelemetryConfig
}

type AppConfig struct {
	ServiceName string `env:"SERVICE_NAME"`
	Version     string `env:"VERSION"`
}

type ProbeConfig struct {
	Address string `env:"PROBE_HTTP_ADDRESS" required:"true"`
	Port    int    `env:"PROBE_HTTP_PORT" required:"true"`
}

type ServicesConfig struct {
	ClockEmuGrpcUrl   string `env:"CLOCK_EMU_GRPC_URL,required"`
	ClockEmuHealthUrl string `env:"CLOCK_EMU_HEALTH_URL,required"`
	CottageManagerUrl string `env:"COTTAGE_MANAGER_URL,required"`
	GuestManagerUrl   string `env:"GUEST_MANAGER_URL,required"`
}

type BookingMachineConfig struct {
	GraphFile                 string `env:"BOOKING_MACHINE_GRAPH_FILE" default:"docs/booking_mdp.dot"`
	ConcurrencyLevel          int    `env:"BOOKING_MACHINE_CONCURRENCY_LEVEL,required"`
	TimeBetweenStepsInSeconds int    `env:"BOOKING_MACHINE_TIME_BETWEEN_STEPS_IN_SECONDS,required"`
}

type GuestRegisterMachineConfig struct {
	GraphFile                 string `env:"GUEST_REGISTER_GRAPH_FILE" default:"docs/guest_register_mdp.dot"`
	ConcurrencyLevel          int    `env:"GUEST_REGISTER_MACHINE_CONCURRENCY_LEVEL,required"`
	TimeBetweenStepsInSeconds int    `env:"GUEST_REGISTER_TIME_BETWEEN_STEPS_IN_SECONDS,required"`
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
		slog.Error("parse env config", "error", err)
		return cfg, err
	}
	return cfg, nil
}
