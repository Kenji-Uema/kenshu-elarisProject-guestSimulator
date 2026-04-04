package telemetry

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"go.opentelemetry.io/contrib/propagators/b3"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

type Config struct {
	Endpoint string
	GrpcPort int
	Insecure bool
}

type AppInfo struct {
	ServiceName string
	Version     string
}

func Init(ctx context.Context, cfg Config, appCfg AppInfo) (func(context.Context) error, error) {
	otelResource, err := resource.New(ctx,
		resource.WithTelemetrySDK(),
		resource.WithAttributes(semconv.ServiceName(fmt.Sprintf("%s:%s", appCfg.ServiceName, appCfg.Version))))
	if err != nil {
		slog.ErrorContext(ctx, "create otel resource", "error", err)
		return nil, err
	}

	traceProvider, err := newTraceProvider(ctx, otelResource, cfg)
	if err != nil {
		slog.ErrorContext(ctx, "create trace provider", "error", err)
		return nil, err
	}

	meterProvider, err := newMeterProvider(ctx, otelResource, cfg)
	if err != nil {
		slog.ErrorContext(ctx, "create meter provider", "error", err)
		return nil, err
	}

	otel.SetTracerProvider(traceProvider)
	otel.SetMeterProvider(meterProvider)

	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			b3.New(),
			propagation.TraceContext{},
			propagation.Baggage{},
		),
	)

	if err := initMetrics(); err != nil {
		slog.ErrorContext(ctx, "create startup histogram", "error", err)
		return nil, err
	}

	return func(ctx context.Context) error {
		shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		var shutdownErr error
		if err := traceProvider.Shutdown(shutdownCtx); err != nil {
			shutdownErr = errors.Join(shutdownErr, err)
		}
		if err := meterProvider.Shutdown(shutdownCtx); err != nil {
			shutdownErr = errors.Join(shutdownErr, err)
		}

		return shutdownErr
	}, nil
}

func newTraceProvider(ctx context.Context, otelResource *resource.Resource, cfg Config) (*sdktrace.TracerProvider, error) {
	exporter, err := otlptracegrpc.New(ctx, otelTraceOptions(cfg)...)
	if err != nil {
		return nil, err
	}

	return sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(otelResource),
	), nil
}

func newMeterProvider(ctx context.Context, otelResource *resource.Resource, cfg Config) (*sdkmetric.MeterProvider, error) {
	exporter, err := otlpmetricgrpc.New(ctx, otelMetricOptions(cfg)...)
	if err != nil {
		return nil, err
	}

	reader := sdkmetric.NewPeriodicReader(exporter)
	return sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(otelResource),
		sdkmetric.WithReader(reader)), nil
}

func otelTraceOptions(cfg Config) []otlptracegrpc.Option {
	opts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(fmt.Sprintf("%s:%d", cfg.Endpoint, cfg.GrpcPort)),
	}
	if cfg.Insecure {
		opts = append(opts, otlptracegrpc.WithInsecure())
	}
	return opts
}

func otelMetricOptions(cfg Config) []otlpmetricgrpc.Option {
	opts := []otlpmetricgrpc.Option{
		otlpmetricgrpc.WithEndpoint(fmt.Sprintf("%s:%d", cfg.Endpoint, cfg.GrpcPort)),
	}
	if cfg.Insecure {
		opts = append(opts, otlpmetricgrpc.WithInsecure())
	}
	return opts
}
