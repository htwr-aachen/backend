package instrumentation

import (
	"context"
	"testing"

	"github.com/htwr-aachen/backend/pkg/config"
)

func TestIntegration_Lifecycle(t *testing.T) {
	cfg := &config.OpenTelemetry{
		Enabled:     false,
		ServiceName: "test-service",
		Endpoint:    "http://localhost:4318",
	}

	ctx := context.Background()
	im, err := Start(ctx, cfg, "test-service")
	if err != nil {
		t.Fatalf("start failed: %v", err)
	}

	err = im.Shutdown(ctx)
	if err != nil {
		t.Fatalf("shutdown failed: %v", err)
	}
}

func TestIntegration_ContextFlow(t *testing.T) {
	cfg := &config.OpenTelemetry{Enabled: false}
	im, _ := Start(context.Background(), cfg, "test")
	ctx := AttachToContext(context.Background(), im)

	retrieved, ok := FromContext(ctx)
	if !ok || retrieved == nil {
		t.Error("context propagation failed")
	}
}
