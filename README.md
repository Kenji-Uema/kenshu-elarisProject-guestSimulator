# guestSimulator

Runs the end-to-end guest journey: registration, booking, payment, and lodging.

## What It Does

- creates simulated guest identities and initializes per-guest state in Redis
- registers the guest in `guestManager`
- books a cottage through `cottageManager`
- sets up and consumes the per-guest RabbitMQ communication queue
- pays invoices through `paymentSimulator`
- drives the lodging WebSocket counterpart of `guestManager`

## Dependencies And Integrations

- HTTP clients to `guestManager`, `cottageManager`, and `paymentSimulator`
- WebSocket client to `guestManager`
- RabbitMQ consumers for guest communication and time events
- Redis for cached journey state
- gRPC client to `clockSimulator`

## Local Commands

```sh
go run ./internal
go build ./internal
make generate
make docker-build
```

There is no dedicated `make test` target in this module at the moment.

## Minimum Env To Start

Optional vars with defaults, such as `SERVICE_NAME`, graph-file paths, Redis DB, and timeout values, are omitted here.

```sh
PROBE_HTTP_ADDRESS=0.0.0.0
PROBE_HTTP_PORT=8080

RABBITMQ_USERNAME=<rabbit user>
RABBITMQ_PASSWORD=<rabbit password>
RABBITMQ_HOST=<rabbit host>
RABBITMQ_PORT=5672

REDIS_HOST=<redis host>
REDIS_PORT=6379

CLOCK_EMU_GRPC_URL=<clock host>
CLOCK_EMU_GRPC_PORT=50051
COTTAGE_MANAGER_URL=<cottage-manager host>
COTTAGE_MANAGER_PORT=8080
GUEST_MANAGER_URL=<guest-manager host>
GUEST_MANAGER_PORT=8080
PAYMENT_SIMULATOR_URL=<payment-simulator host>
PAYMENT_SIMULATOR_PORT=8080

HOUR_CHANGE_QUEUE_NAME=q.guest-simulator.hour-change
HOUR_CHANGE_BINDING_EXCHANGE_NAME=ex.time.event
HOUR_CHANGE_BINDING_ROUTING_KEY=time.event.hour

COMMUNICATION_BINDING_EXCHANGE_NAME=ex.communication

BOOKING_FLOW_CONCURRENCY_LEVEL=2
LODGING_FLOW_CONCURRENCY_LEVEL=2
PAYMENT_FLOW_CONCURRENCY_LEVEL=2
GUEST_REGISTER_FLOW_CONCURRENCY_LEVEL=2
GUEST_JOURNEY_FLOW_CONCURRENCY_LEVEL=2

OTEL_EXPORTER_OTLP_ENDPOINT=<otel host>
OTEL_EXPORTER_OTLP_GRPC_PORT=4317
OTEL_EXPORTER_OTLP_HEALTH_PORT=13133
OTEL_EXPORTER_OTLP_INSECURE=true
```

## Configuration

Configuration is environment-driven. Start with:

- `internal/config/config.go`

Important groups:

- probe HTTP: `PROBE_HTTP_*`
- downstream services: `CLOCK_EMU_*`, `COTTAGE_MANAGER_*`, `GUEST_MANAGER_*`, `PAYMENT_SIMULATOR_*`
- RabbitMQ connection plus consumer settings: `RABBITMQ_*`, guest communication, time-event
- Redis: `REDIS_*`
- flow tuning: `BOOKING_FLOW_*`, `GUEST_REGISTER_*`, `LODGING_FLOW_*`, `PAYMENT_FLOW_*`, `GUEST_JOURNEY_*`
- telemetry: `OTEL_EXPORTER_OTLP_*`

## Key Files

- `internal/main.go`
- `internal/app/journey/guest_journey.go`
- `internal/app/flows/`
- `internal/app/steps/lodging_step/`
- `internal/config/`
