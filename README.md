# Guest Simulator

Runs the end-to-end guest journey: registration, booking, payment, and lodging.

## Main Docs

See the main project documentation: <https://kenji-uema.github.io/kenshu-elarisProject-docs/>

## What It Does

- creates simulated guest identities and initializes per-guest state in Redis
- registers the guest in `guestManager`
- books a cottage through `cottageManager`
- sets up and consumes the per-guest RabbitMQ communication queue
- pays invoices through `paymentSimulator`
- drives the lodging WebSocket counterpart of `guestManager`

## Interfaces And Dependencies

- HTTP clients to `guestManager`, `cottageManager`, and `paymentSimulator`
- WebSocket client to `guestManager`
- RabbitMQ consumers for guest communication and time events
- Redis for cached journey state
- gRPC client to `clockSimulator`

## RabbitMQ Specification

This service consumes two RabbitMQ streams:

1. a shared hour-change stream used to advance time-driven behavior
2. a per-guest communication stream used during booking, payment, and lodging

### Connection

Use the standard AMQP connection env vars:

```sh
RABBITMQ_USERNAME=<rabbit user>
RABBITMQ_PASSWORD=<rabbit password>
RABBITMQ_HOST=<rabbit host>
RABBITMQ_PORT=5672
```

### Consumer Topology

#### 1. Hour Change Consumer

- Queue: `HOUR_CHANGE_QUEUE_NAME` default example `q.guest-simulator.hour-change`
- Exchange: `HOUR_CHANGE_BINDING_EXCHANGE_NAME` default example `ex.time.event`
- Routing key: `HOUR_CHANGE_BINDING_ROUTING_KEY` default example `time.event.hour`
- Queue behavior: durable by default unless overridden by env
- Ack mode: manual ack
- Payload encoding: protobuf `event.TimeEvent`

This queue is declared on startup and remains shared across the service instance.

#### 2. Guest Communication Consumer

- Exchange: `COMMUNICATION_BINDING_EXCHANGE_NAME` default/fallback `ex.communication`
- Queue name pattern: `q.guest.<guestId>`
- Routing key pattern: `guest.<guestId>`
- Queue behavior: forced to `durable=false` and `autoDelete=true`
- Ack mode: manual ack
- Payload encoding: protobuf
- Message selector: AMQP header `message_type`

This queue is created after the guest is registered and is dedicated to a single simulated guest journey. It is auto-deleted when the consumer is cleaned up.

### Consumed Queues And Messages

#### Hour Change Queue

The service consumes `event.TimeEvent` messages from the configured hour-change queue.

Expected protobuf shape:

```proto
message TimeEvent {
  google.protobuf.Timestamp time = 1;
}
```

Decoded example:

```json
{
  "time": "2026-04-12T15:00:00Z"
}
```

Behavior:

- valid messages are acked and broadcast internally to time subscribers
- invalid protobuf payloads are nacked with `requeue=false`

#### Guest Communication Queue

The service consumes messages from `q.guest.<guestId>` and dispatches them by the `message_type` AMQP header.

Required AMQP metadata:

```text
exchange: ex.communication
routing_key: guest.<guestId>
headers.message_type: <contract-specific type>
body: protobuf-encoded payload
```

Supported `message_type` values:

- `paymentSimulator.payment.v1.PaymentRequest`
- `cottageManager.invoice.BookingConfirmedNotificationEvent`
- `lodging.v1.CheckInTodayNotification`

Any other `message_type` is treated as unsupported and the delivery is nacked with `requeue=false`.

### Message Contracts

#### Payment Request

- Header `message_type`: `paymentSimulator.payment.v1.PaymentRequest`
- Protobuf message: `communication.v1.PaymentRequest`
- Used by the payment stage after a booking is created

Relevant fields:

```proto
message PaymentRequest {
  string invoice_number = 1;
  string description = 2;
  Money total = 3;
  google.protobuf.Timestamp issued_at = 4;
  google.protobuf.Timestamp expires_at = 5;
  BookingSummary booking = 6;
  PayerSummary payer = 7;
  repeated PaymentOption options = 8;
}
```

Decoded example:

```json
{
  "invoice_number": "INV-2026-000123",
  "description": "Booking payment for cottage stay",
  "total": {
    "amount": 420000,
    "currency": "BRL"
  },
  "issued_at": "2026-04-12T15:03:00Z",
  "expires_at": "2026-04-12T18:03:00Z",
  "booking": {
    "cottage_name": "Serra Azul Cabin",
    "nights": 3,
    "number_of_guests": 2
  },
  "payer": {
    "name": "Guest Simulator",
    "email": "guest.simulator@test.com"
  },
  "options": [
    {
      "method": "pix",
      "payment_url": "https://payment-simulator/pay/INV-2026-000123",
      "instructions": "Pay before expiration"
    }
  ]
}
```

Matching rules used by `guestSimulator`:

- `invoice_number` must be present
- `total.amount` must be positive and `total.currency` must be present
- `booking.cottage_name` must match the selected cottage in cache
- `payer.email` must match the simulated guest email in cache

#### Booking Confirmation

- Header `message_type`: `cottageManager.invoice.BookingConfirmedNotificationEvent`
- Protobuf message: `communication.v1.BookingConfirmedNotificationEvent`
- Used after payment to confirm the booking transitioned to `CONFIRMED`

Relevant fields:

```proto
message BookingConfirmedNotificationEvent {
  string id = 1;
  string booking_id = 2;
  BookingStatus booking_status = 4;
  Guest guest = 5;
  BookingConfirmationSummary booking = 6;
  google.protobuf.Timestamp confirmed_at = 8;
}
```

Decoded example:

```json
{
  "id": "evt-booking-confirmed-001",
  "booking_id": "booking-123",
  "booking_status": "BOOKING_STATUS_CONFIRMED",
  "guest": {
    "guest_id": "guest-123",
    "name": "Guest Simulator",
    "email": "guest.simulator@test.com",
    "phone": "+55-11-99999-9999"
  },
  "booking": {
    "cottage_name": "Serra Azul Cabin",
    "check_in": "2026-04-20T15:00:00Z",
    "check_out": "2026-04-23T11:00:00Z",
    "nights": 3,
    "number_of_guests": 2
  },
  "confirmed_at": "2026-04-12T15:05:00Z"
}
```

Matching rules used by `guestSimulator`:

- `booking_id` must match the cached booking id
- `booking_status` must be `BOOKING_STATUS_CONFIRMED`
- `guest.guest_id` must match the active simulated guest id

#### Check-In Today Notification

- Header `message_type`: `lodging.v1.CheckInTodayNotification`
- Protobuf message: `lodging.v1.CheckInTodayNotification`
- Used to unblock the lodging stage

Relevant fields:

```proto
message CheckInTodayNotification {
  string booking_id = 1;
  string guest_id = 2;
  string cottage_name = 3;
  google.protobuf.Timestamp check_in = 4;
  google.protobuf.Timestamp check_out = 5;
  int32 number_of_guests = 6;
  google.protobuf.Timestamp notification_day = 7;
}
```

Decoded example:

```json
{
  "booking_id": "booking-123",
  "guest_id": "guest-123",
  "cottage_name": "Serra Azul Cabin",
  "check_in": "2026-04-20T15:00:00Z",
  "check_out": "2026-04-23T11:00:00Z",
  "number_of_guests": 2,
  "notification_day": "2026-04-19T09:00:00Z"
}
```

Matching rules used by `guestSimulator`:

- `booking_id` must match the cached booking id
- `guest_id` must match the active simulated guest id
- `cottage_name` must match the selected cottage
- `check_in` must match the selected booking start date

### Delivery Semantics

- RabbitMQ consumers are created with `AUTO_ACK=false` by default
- supported and successfully parsed messages are acked
- invalid payloads or unsupported `message_type` values are nacked with `requeue=false`
- guest communication messages can be buffered in-memory until the relevant journey step subscribes

### Relevant Configuration

```sh
HOUR_CHANGE_QUEUE_NAME=q.guest-simulator.hour-change
HOUR_CHANGE_BINDING_EXCHANGE_NAME=ex.time.event
HOUR_CHANGE_BINDING_ROUTING_KEY=time.event.hour

COMMUNICATION_BINDING_EXCHANGE_NAME=ex.communication
COMMUNICATION_CONSUME_AUTO_ACK=false
```

## WebSocket Lodging Chat

The lodging stage connects to `guestManager` over WebSocket and simulates the guest side of the resort stay lifecycle.

### Endpoint

The client connects to:

```text
ws://<GUEST_MANAGER_URL>:<GUEST_MANAGER_PORT>/lodging/chat
```

In code, the URL is built from:

- `GUEST_MANAGER_URL`
- `GUEST_MANAGER_PORT`

### Transport Format

- WebSocket message type: text
- Serialization: protobuf message `lodging.v1.ChatMessage` encoded as JSON via `protojson`
- Protocol version sent by this client: `lodging.v1`
- Trace propagation: OpenTelemetry trace context is copied into `trace_context`

Envelope:

```proto
message ChatMessage {
  string message_id = 1;
  string correlation_id = 2;
  Sender sender = 4;
  string protocol_version = 12;
  map<string, string> trace_context = 13;

  oneof payload {
    GuestAction guest_action = 6;
    GuestResponse guest_response = 7;
    SystemNotification system_notification = 8;
    SystemRequest system_request = 9;
    Ack ack = 10;
  }
}
```

### Acknowledgment Behavior

- every guest-originated action or response waits for an inbound `ack`
- every non-ack message received from the server is automatically acknowledged by `guestSimulator`
- the automatic ack uses:
  - `sender = SENDER_GUEST`
  - `protocol_version = "lodging.v1"`
  - `ack.acknowledged_message_id = <received message_id>`
  - `ack.status = ACK_STATUS_ACCEPTED`
  - `ack.code = ERROR_CODE_NONE`

Ack example:

```json
{
  "messageId": "c3d09b2d-1c9e-4f0f-a8d5-6f0cb6a6f7ad",
  "correlationId": "srv-msg-001",
  "sender": "SENDER_GUEST",
  "protocolVersion": "lodging.v1",
  "payload": {
    "ack": {
      "acknowledgedMessageId": "srv-msg-001",
      "status": "ACK_STATUS_ACCEPTED",
      "code": "ERROR_CODE_NONE"
    }
  }
}
```

### Message Types Used By The Guest Client

#### Guest Actions Sent

The default lodging flow can emit these `GuestAction` values:

- `SHOW_FOR_CHECKIN`
- `TAKE_COTTAGE_KEY`
- `ENTER_COTTAGE`
- `GO_FOR_A_BATH`
- `GO_FOR_DINNER`
- `GO_TO_SLEEP`
- `WAKEUP`
- `GO_FOR_BREAKFAST`
- `LEAVE_CLEANUP_NOTIFICATION`
- `ENJOY_RESORT`
- `LEAVE_COTTAGE`
- `PROCEED_TO_CHECKOUT`
- `RETURN_COTTAGE_KEY`

Guest action example:

```json
{
  "messageId": "9c41ec3a-7d3a-4798-b4c2-8b98de2a5b4a",
  "correlationId": "b3d2c2ea-9c8f-4227-95c8-13c22dc8f0b5",
  "sender": "SENDER_GUEST",
  "protocolVersion": "lodging.v1",
  "payload": {
    "guestAction": "SHOW_FOR_CHECKIN"
  }
}
```

#### Guest Responses Sent

The client replies to specific system requests using `GuestResponse`:

- `REQUEST_DOCUMENT` -> `show_document.document_id`
- `REQUEST_BOOKING_NUMBER` -> `show_booking_number.booking_id`
- `GIVE_COTTAGE_KEY` -> `receive_cottage_key.cottage_key_id`
- `REQUEST_COTTAGE_KEY` -> `return_cottage_key.cottage_key_id`

Response example for document submission:

```json
{
  "messageId": "0f756cdb-7a2e-45ff-87d5-0fbbfcf0d51a",
  "correlationId": "srv-request-001",
  "sender": "SENDER_GUEST",
  "protocolVersion": "lodging.v1",
  "payload": {
    "guestResponse": {
      "showDocument": {
        "documentId": "123-45-6789"
      }
    }
  }
}
```

#### System Messages Expected From The Server

System requests consumed by the client:

- `REQUEST_DOCUMENT`
- `REQUEST_BOOKING_NUMBER`
- `GIVE_COTTAGE_KEY`
- `REQUEST_COTTAGE_KEY`

System notifications consumed by the client:

- `BOOKING_CHECKING`
- `CHECK_IN_COMPLETE`
- `DINNER_READY`
- `BREAKFAST_READY`
- `CHECK_OUT_COMPLETE`

Other notifications defined in the protobuf may exist, but these are the ones the default guest flow explicitly waits for.

Server request example:

```json
{
  "messageId": "srv-request-001",
  "correlationId": "checkin-guest-123",
  "sender": "SENDER_SYSTEM",
  "protocolVersion": "lodging.v1",
  "payload": {
    "systemRequest": "REQUEST_DOCUMENT"
  }
}
```

Server notification example:

```json
{
  "messageId": "srv-notification-001",
  "correlationId": "checkin-guest-123",
  "sender": "SENDER_SYSTEM",
  "protocolVersion": "lodging.v1",
  "payload": {
    "systemNotification": "CHECK_IN_COMPLETE"
  }
}
```

### Default Dialogue Flow

The lodging step uses the cached guest and booking data plus hour-change events to drive the chat.

Check-in phase:

1. Wait until check-in day at or after `15:00 UTC`
2. Send `SHOW_FOR_CHECKIN`
3. Respond to `REQUEST_DOCUMENT` with the cached guest document id
4. Respond to `REQUEST_BOOKING_NUMBER` with the cached booking id
5. Wait for `CHECK_IN_COMPLETE`
6. Wait for `GIVE_COTTAGE_KEY`
7. Reply with `receive_cottage_key`
8. Send `TAKE_COTTAGE_KEY`

Stay phase:

1. Send `ENTER_COTTAGE`
2. Follow the configured schedule using hour-change notifications
3. Wait for `DINNER_READY` before `GO_FOR_DINNER`
4. Wait for `BREAKFAST_READY` before `GO_FOR_BREAKFAST`

Checkout phase:

1. Wake up on checkout day
2. Leave cottage and proceed to checkout
3. Wait for `REQUEST_COTTAGE_KEY`
4. Reply with `return_cottage_key`
5. Send `RETURN_COTTAGE_KEY`
6. Wait for `CHECK_OUT_COMPLETE`

### Operational Notes

- unrelated inbound system messages are ignored until the expected request or notification arrives
- inbound non-ack messages are acknowledged immediately before being dispatched internally
- outbound messages use generated UUIDs for `message_id`
- replies reuse the inbound request `correlation_id`
- messages received while the client is waiting for an ack are buffered in-memory and processed afterward

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
