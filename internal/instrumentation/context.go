package instrumentation

import "context"

type contextKey string

const instrumentationKey contextKey = "instrumentation"

// AttachToContext attaches InstrumentationManager to context
func AttachToContext(ctx context.Context, im *InstrumentationManager) context.Context {
	return context.WithValue(ctx, instrumentationKey, im)
}

// FromContext retrieves InstrumentationManager from context
func FromContext(ctx context.Context) (*InstrumentationManager, bool) {
	im, ok := ctx.Value(instrumentationKey).(*InstrumentationManager)
	return im, ok
}
