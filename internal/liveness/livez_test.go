package liveness

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLivenessServer_LivezHandler(t *testing.T) {
	t.Run("Returns 200 OK", func(t *testing.T) {
		rm := NewReadinessManager()
		s := NewLivenessServer(rm)

		req := httptest.NewRequest(http.MethodGet, "/livez", nil)
		w := httptest.NewRecorder()

		s.LivezHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}
	})

	t.Run("Returns correct JSON", func(t *testing.T) {
		rm := NewReadinessManager()
		s := NewLivenessServer(rm)

		req := httptest.NewRequest(http.MethodGet, "/livez", nil)
		w := httptest.NewRecorder()

		s.LivezHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		expected := `{"status":"ok"}`

		if string(body) != expected {
			t.Errorf("expected body '%s', got '%s'", expected, string(body))
		}
	})
}
