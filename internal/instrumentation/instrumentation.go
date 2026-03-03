package instrumentation

import (
	"context"
	"fmt"

	"github.com/htwr-aachen/backend/pkg/config"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
)

// InstrumentationManager manages OpenTelemetry resources
type InstrumentationManager struct {
	tracerProvider *sdktrace.TracerProvider
	shutdown       func(context.Context) error
}

// Start initializes OpenTelemetry based on configuration
func Start(ctx context.Context, cfg *config.OpenTelemetry, serviceName string) (*InstrumentationManager, error) {
	if !cfg.Enabled {
		return &InstrumentationManager{
			shutdown: func(ctx context.Context) error { return nil },
		}, nil
	}

	exporter, err := otlptracehttp.New(ctx, otlptracehttp.WithEndpoint(cfg.Endpoint))
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(cfg.ServiceName),
			semconv.ServiceVersionKey.String("1.0.0"),
		),
	)
	if err != nil {
		exporter.Shutdown(ctx)
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	otel.SetTracerProvider(tracerProvider)

	return &InstrumentationManager{
		tracerProvider: tracerProvider,
		shutdown: func(ctx context.Context) error {
			return tracerProvider.Shutdown(ctx)
		},
	}, nil
}

func (im *InstrumentationManager) Shutdown(ctx context.Context) error {
	if im.shutdown == nil {
		return nil
	}
	return im.shutdown(ctx)
}

func (im *InstrumentationManager) GetTracer(scopeName string, opts ...trace.TracerOption) trace.Tracer {
	if im.tracerProvider == nil {
		return otel.Tracer(scopeName, opts...)
	}
	return im.tracerProvider.Tracer(scopeName, opts...)
}
