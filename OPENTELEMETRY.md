# OpenTelemetry Integration

This guide explains how OpenTelemetry has been integrated into the HTWR backend server.

## Overview

OpenTelemetry provides distributed tracing capabilities to track requests across your system. The integration includes:

- **Trace Exporting**: Sends traces to an OTLP-compatible collector (e.g., Jaeger, OpenTelemetry Collector)
- **HTTP Tracing**: Automatically traces all incoming HTTP requests
- **Configurable**: Can be enabled/disabled via configuration

## Architecture

### Components

1. **instrumentation package** (`internal/instrumentation/`)
   - `instrumentation.go`: Core OpenTelemetry initialization
   - `context.go`: Context attachment/retrieval helpers
   - `http.go`: HTTP tracing middleware wrapper

2. **Configuration** (`pkg/config/opentelemetry.go`)
   - `Enabled`: Enable/disable OpenTelemetry
   - `ServiceName`: Service identifier (default: "htwr-backend")
   - `Endpoint`: OTLP HTTP collector endpoint

3. **Server Integration** (`internal/server/`)
   - HTTP handlers wrapped with `otelhttp.NewHandler()`
   - Automatic span creation for each request

## Configuration

### Environment Variables

```
HTWR_BACKEND_GLOBAL_OPENTELEMETRY_ENABLED=true
HTWR_BACKEND_GLOBAL_OPENTELEMETRY_ENDPOINT=http://localhost:4318
HTWR_BACKEND_GLOBAL_OPENTELEMETRY_SERVICE_NAME=htwr-backend
```

### YAML Configuration File

```yaml
global:
  opentelemetry:
    enabled: true
    service_name: htwr-backend
    endpoint: http://localhost:4318
```

### Default Values

- **Enabled**: `false` (disabled by default)
- **Service Name**: `htwr-backend`
- **Endpoint**: `http://localhost:4318` (standard OTLP HTTP port)

## Usage

### Enable OpenTelemetry

```bash
# Via environment variable
export HTWR_BACKEND_GLOBAL_OPENTELEMETRY_ENABLED=true
./htwr-backend run

# Via command line (if you add a flag for it)
./htwr-backend run --opentelemetry-enabled=true

# Via config file
./htwr-backend run --config config.yaml
```

### Setup OTLP Collector

#### Using Docker Compose

```yaml
version: '3'
services:
  otel-collector:
    image: otel/opentelemetry-collector:0.84.0
    command: ["--config=/etc/otel-collector-config.yaml"]
    ports:
      - "4317:4317"   # OTLP gRPC
      - "4318:4318"   # OTLP HTTP
    volumes:
      - ./otel-config.yaml:/etc/otel-collector-config.yaml

  jaeger:
    image: jaegertracing/all-in-one:1.39
    ports:
      - "16686:16686" # Jaeger UI
    environment:
      - COLLECTOR_OTLP_ENABLED=true
```

#### OpenTelemetry Collector Config (otel-config.yaml)

```yaml
receivers:
  otlp:
    protocols:
      http:
        endpoint: "0.0.0.0:4318"
      grpc:
        endpoint: "0.0.0.0:4317"

processors:
  batch:
  memory_limiter:
    check_interval: 1s
    limit_mib: 512

exporters:
  jaeger:
    endpoint: "http://localhost:14250"
    tls:
      insecure: true

service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [memory_limiter, batch]
      exporters: [jaeger]
```

## Tracing Routes

All HTTP routes are automatically traced:

- **Public API** (`/api/qa/*`, `/api/panikzettel/*`): Traced under "public-api"
- **Admin API** (`/`): Traced under "admin-api"
- **Metrics** (`/metrics`, `/livez`, `/readyz`): Traced under "metrics"

## Viewing Traces

Once configured with Jaeger:

1. Open Jaeger UI: `http://localhost:16686`
2. Select service: "htwr-backend"
3. View traces from your requests

## Adding Custom Spans

To add custom spans in your code:

```go
package mypackage

import (
	"context"
	"github.com/htwr-aachen/backend/internal/instrumentation"
	"go.opentelemetry.io/otel"
)

func MyFunction(ctx context.Context) {
	// Get instrumentation manager from context
	im, ok := instrumentation.FromContext(ctx)
	if !ok {
		// OpenTelemetry not initialized
		return
	}
	
	// Get tracer
	tracer := im.GetTracer("mypackage")
	
	// Create a span
	ctx, span := tracer.Start(ctx, "my-operation")
	defer span.End()
	
	// Your code here
}

// Or use the global tracer
func MyFunction(ctx context.Context) {
	tracer := otel.Tracer("mypackage")
	ctx, span := tracer.Start(ctx, "my-operation")
	defer span.End()
	
	// Your code here
}
```

## Performance Considerations

- **Batching**: Traces are batched before export for efficiency
- **Resource Memory**: Default memory limit is 512MB
- **Disabled by Default**: OpenTelemetry doesn't impact performance when disabled

## Production Deployment

### 1. Use a Production Collector

Instead of Jaeger all-in-one, use a dedicated OpenTelemetry Collector with appropriate backends:

```yaml
exporters:
  otlp:
    endpoint: "otlp-backend:4317"
  prometheus:
    endpoint: "0.0.0.0:8888"
```

### 2. Set Appropriate Endpoints

```bash
export HTWR_BACKEND_GLOBAL_OPENTELEMETRY_ENDPOINT=http://otel-collector.prod:4318
```

### 3. Configure Resource Limits

Adjust memory limits based on your trace volume in the collector config.

### 4. Implement Sampling

To reduce trace volume in production, consider implementing sampling (can be added to instrumentation):

```go
// In instrumentation.go
sdktrace.NewTracerProvider(
    sdktrace.WithSampler(
        sdktrace.ParentBased(
            sdktrace.TraceIDRatioBased(0.1), // Sample 10% of traces
        ),
    ),
    // ... other options
)
```

## Troubleshooting

### No Traces Appearing

1. Verify OpenTelemetry is enabled: Check logs for "OpenTelemetry initialized"
2. Verify collector is running: `curl http://localhost:4318/healthcheck`
3. Check endpoint configuration matches collector address
4. Check firewall/network connectivity to collector

### Memory Usage High

- Reduce trace sampling rate
- Lower memory limit in collector config
- Check for batch processor configuration

### Missing Traces

- Verify resource name matches in Jaeger service filter
- Check if sampling is configured
- Verify batch processor isn't dropping traces

## Dependencies

The integration uses:
- `go.opentelemetry.io/otel`: Core OpenTelemetry API
- `go.opentelemetry.io/otel/sdk`: SDK for tracing
- `go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp`: OTLP HTTP exporter
- `go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp`: HTTP instrumentation

## Further Reading

- [OpenTelemetry Documentation](https://opentelemetry.io/docs/)
- [OTLP Protocol](https://opentelemetry.io/docs/reference/protocol/)
- [Jaeger Documentation](https://www.jaegertracing.io/docs/)
- [OpenTelemetry Go](https://opentelemetry.io/docs/instrumentation/go/)
