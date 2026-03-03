package instrumentation

import (
	"context"
	"testing"
)

func TestAttachContext(t *testing.T) {
	im := &InstrumentationManager{}
	ctx := context.Background()
	newCtx := AttachToContext(ctx, im)

	if ctx == newCtx {
		t.Error("context should be different")
	}
}

func TestFromContextSuccess(t *testing.T) {
	im := &InstrumentationManager{}
	ctx := AttachToContext(context.Background(), im)

	retrieved, ok := FromContext(ctx)
	if !ok {
		t.Error("failed to retrieve from context")
	}
	if retrieved != im {
		t.Error("retrieved manager doesn't match")
	}
}

func TestFromContextEmpty(t *testing.T) {
	_, ok := FromContext(context.Background())
	if ok {
		t.Error("should return false for empty context")
	}
}
