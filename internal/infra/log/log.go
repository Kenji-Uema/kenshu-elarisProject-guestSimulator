package log

import (
	"context"
	"log/slog"
	"os"

	"github.com/Kenji-Uema/guestSimulator/internal/config"
	"go.opentelemetry.io/otel/trace"
)

type TraceHandler struct {
	next slog.Handler
}

func NewLogger(config config.AppConfig) *slog.Logger {
	base := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:     slog.LevelInfo,
		AddSource: true,
	})

	h := &TraceHandler{next: base}

	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	return slog.New(h).With(
		"app", hostname,
		"service.name", config.Name.ServiceName,
		"service.version", config.Name.Version,
		"service.namespace", config.Name.ServiceNamespace,
		"service.instance.id", hostname,
	)
}

func (h *TraceHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

func (h *TraceHandler) Handle(ctx context.Context, r slog.Record) error {
	sc := trace.SpanFromContext(ctx).SpanContext()
	if sc.IsValid() {
		r.AddAttrs(
			slog.String("trace_id", sc.TraceID().String()),
			slog.String("span_id", sc.SpanID().String()),
		)
	}
	return h.next.Handle(ctx, r)
}

func (h *TraceHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &TraceHandler{next: h.next.WithAttrs(attrs)}
}

func (h *TraceHandler) WithGroup(name string) slog.Handler {
	return &TraceHandler{next: h.next.WithGroup(name)}
}
