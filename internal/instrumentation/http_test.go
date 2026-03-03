package instrumentation

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTPHandlerWrapping(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	wrapped := WrapHTTPHandler(handler, "test-op")
	if wrapped == nil {
		t.Error("wrapped handler is nil")
	}

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	wrapped.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, w.Code)
	}
}

func TestHTTPHandlerFuncWrapping(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}

	wrapped := WrapHTTPHandlerFunc(http.HandlerFunc(handler), "test-op")
	if wrapped == nil {
		t.Error("wrapped handler is nil")
	}

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	wrapped.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, w.Code)
	}
}
