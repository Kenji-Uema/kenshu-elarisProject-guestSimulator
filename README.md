# guestSimulator

Drives guest-side simulations for registration, booking, payment, and lodging.

## Responsibilities

- simulate guest registration
- simulate cottage booking
- simulate payment completion through `paymentSimulator`
- drive the guest counterpart of the lodging WebSocket flow
- react to hour-change events from RabbitMQ

## Interfaces

- HTTP clients to guest and cottage services
- gRPC client to `clockSimulator`
- HTTP client to `paymentSimulator`
- WebSocket client to `guestManager`
- RabbitMQ consumer for hour-change notifications
- Redis connectivity for readiness and runtime infra checks

## Run

```sh
go run ./internal
```

## Build

```sh
make build
make docker-build
```

## Configuration

Configuration is environment-driven. See:

- `internal/config/config.go`

Important families:

- probe HTTP: `PROBE_HTTP_*`
- service endpoints: `CLOCK_EMU_*`, `COTTAGE_MANAGER_*`, `GUEST_MANAGER_*`, `PAYMENT_SIMULATOR_*`
- RabbitMQ: `RABBITMQ_*`, `TIME_EVENT_*`
- Redis: `REDIS_*`
- machine control: `BOOKING_MACHINE_*`, `GUEST_REGISTER_*`, `LODGING_MACHINE_*`
- telemetry: `OTEL_EXPORTER_OTLP_*`

## Docs

State and flow diagrams live in `docs/`.

## Entry points

- `internal/main.go`
- `internal/app/booking_machine.go`
- `internal/app/guest_register_machine.go`
- `internal/app/lodging_machine.go`


SERVICE_NAME=guest-simulator;
VERSION=latest;
PROBE_HTTP_ADDRESS=0.0.0.0;
PROBE_HTTP_PORT=8080;
READ_HEADER_TIMEOUT_IN_SECONDS=5;
READ_TIMEOUT_IN_SECONDS=10;
WRITE_TIMEOUT_IN_SECONDS=15;
IDLE_TIMEOUT_IN_SECONDS=60;
RABBITMQ_USERNAME=guest;
RABBITMQ_PASSWORD=guest;
RABBITMQ_HOST=localhost;
RABBITMQ_PORT=30002;
TIME_EVENT_HOUR_QUEUE_NAME=q.guest-simulator.hour-change;
TIME_EVENT_EXCHANGE_NAME=ex.time.event;
REDIS_HOST=localhost;
REDIS_PORT=30012;
REDIS_DB=0;
CLOCK_EMU_GRPC_URL=localhost;
CLOCK_EMU_GRPC_PORT=30009;
COTTAGE_MANAGER_URL=localhost;
COTTAGE_MANAGER_PORT=30011;
GUEST_MANAGER_URL=localhost;
GUEST_MANAGER_PORT=30010;
PAYMENT_SIMULATOR_URL=localhost;
PAYMENT_SIMULATOR_PORT=30013;
BOOKING_MACHINE_CONCURRENCY_LEVEL=1;
BOOKING_MACHINE_TIME_BETWEEN_STEPS_IN_SECONDS=2;
LODGING_MACHINE_CONCURRENCY_LEVEL=1;
LODGING_MACHINE_TIME_BETWEEN_STEPS_IN_SECONDS=2;
GUEST_REGISTER_MACHINE_CONCURRENCY_LEVEL=1;
GUEST_REGISTER_TIME_BETWEEN_STEPS_IN_SECONDS=2;
PAYMENT_MACHINE_CONCURRENCY_LEVEL=1;
PAYMENT_MACHINE_TIME_BETWEEN_STEPS_IN_SECONDS=2;
GUEST_JOURNEY_MACHINE_CONCURRENCY_LEVEL=1;
GUEST_JOURNEY_TIME_BETWEEN_STEPS_IN_SECONDS=2;
OTEL_EXPORTER_OTLP_ENDPOINT=localhost;
OTEL_EXPORTER_OTLP_GRPC_PORT=30007;
OTEL_EXPORTER_OTLP_HEALTH_PORT=30008;
OTEL_EXPORTER_OTLP_INSECURE=true;



BOOKING_MACHINE_CONCURRENCY_LEVEL=1;BOOKING_MACHINE_TIME_BETWEEN_STEPS_IN_SECONDS=2;
CLOCK_EMU_GRPC_HOST=localhost;CLOCK_EMU_GRPC_PORT=30009;CLOCK_EMU_GRPC_URL=localhost;
COTTAGE_MANAGER_PORT=30011;COTTAGE_MANAGER_URL=localhost;GUEST_MANAGER_PORT=30010;
GUEST_MANAGER_URL=localhost;GUEST_REGISTER_MACHINE_CONCURRENCY_LEVEL=1;
GUEST_REGISTER_TIME_BETWEEN_STEPS_IN_SECONDS=2;OTEL_EXPORTER_OTLP_ENDPOINT=localhost;
OTEL_EXPORTER_OTLP_GRPC_PORT=30007;OTEL_EXPORTER_OTLP_HEALTH_PORT=30008;OTEL_EXPORTER_OTLP_INSECURE=true;
PROBE_HTTP_ADDRESS=localhost;PROBE_HTTP_PORT=8080;RABBITMQ_CONFIRM_NOT_WAIT=false;SERVICE_HOST=localhost;
SERVICE_NAME=guestEmulator;SERVICE_PORT=8080;VERSION=dev