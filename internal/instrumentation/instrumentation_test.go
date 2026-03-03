package instrumentation

import (
	"context"
	"testing"

	"github.com/htwr-aachen/backend/pkg/config"
)

func TestStart_Disabled(t *testing.T) {
	cfg := &config.OpenTelemetry{Enabled: false}
	im, err := Start(context.Background(), cfg, "test-service")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if im == nil {
		t.Error("expected non-nil manager")
	}
}

func TestStart_CustomEndpoint(t *testing.T) {
	cfg := &config.OpenTelemetry{
		Enabled:     true,
		ServiceName: "test-service",
		Endpoint:    "http://localhost:4318",
	}
	im, err := Start(context.Background(), cfg, "test-service")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if im == nil {
		t.Error("expected non-nil manager")
	}
	im.Shutdown(context.Background())
}

func TestShutdown(t *testing.T) {
	cfg := &config.OpenTelemetry{Enabled: false}
	im, _ := Start(context.Background(), cfg, "test-service")
	err := im.Shutdown(context.Background())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGetTracer(t *testing.T) {
	cfg := &config.OpenTelemetry{Enabled: false}
	im, _ := Start(context.Background(), cfg, "test-service")
	tracer := im.GetTracer("test-scope")
	if tracer == nil {
		t.Error("tracer should not be nil")
	}
}
