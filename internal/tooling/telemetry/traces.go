package telemetry

import (
	"go.opentelemetry.io/otel"
)

var Tracer = otel.Tracer("guest-emulator")
