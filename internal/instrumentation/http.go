package instrumentation

import (
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// WrapHTTPHandler wraps an HTTP handler with OpenTelemetry tracing
// This will automatically create spans for each HTTP request
func WrapHTTPHandler(handler http.Handler, operationName string) http.Handler {
	return otelhttp.NewHandler(handler, operationName)
}

// WrapHTTPHandlerFunc wraps an HTTP handler function with OpenTelemetry tracing
func WrapHTTPHandlerFunc(handler http.HandlerFunc, operationName string) http.Handler {
	return otelhttp.NewHandler(handler, operationName)
}
