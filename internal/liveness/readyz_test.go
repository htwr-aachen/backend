package liveness

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestReadinessManager_Register(t *testing.T) {
	t.Run("Register single check", func(t *testing.T) {
		rm := NewReadinessManager()
		check := NewAlwaysHealthyCheck("test-check")

		rm.Register(check)

		if len(rm.checks) != 1 {
			t.Errorf("expected 1 check, got %d", len(rm.checks))
		}
	})

	t.Run("Register multiple checks", func(t *testing.T) {
		rm := NewReadinessManager()
		check1 := NewAlwaysHealthyCheck("check1")
		check2 := NewAlwaysHealthyCheck("check2")

		rm.Register(check1)
		rm.Register(check2)

		if len(rm.checks) != 2 {
			t.Errorf("expected 2 checks, got %d", len(rm.checks))
		}
	})

	t.Run("Replace existing check with same name", func(t *testing.T) {
		rm := NewReadinessManager()
		check1 := NewAlwaysHealthyCheck("test")
		check2 := NewAlwaysUnhealthyCheck("test")

		rm.Register(check1)
		rm.Register(check2)

		if len(rm.checks) != 1 {
			t.Errorf("expected 1 check, got %d", len(rm.checks))
		}

		// Verify it's the second check
		result := rm.checks["test"].Check(context.Background())
		if result.Status != StatusUnhealthy {
			t.Error("expected replaced check to be unhealthy")
		}
	})
}

func TestReadinessManager_Unregister(t *testing.T) {
	t.Run("Unregister existing check", func(t *testing.T) {
		rm := NewReadinessManager()
		check := NewAlwaysHealthyCheck("test-check")

		rm.Register(check)
		rm.Unregister("test-check")

		if len(rm.checks) != 0 {
			t.Errorf("expected 0 checks, got %d", len(rm.checks))
		}
	})

	t.Run("Unregister non-existent check", func(t *testing.T) {
		rm := NewReadinessManager()

		// Should not panic
		rm.Unregister("non-existent")

		if len(rm.checks) != 0 {
			t.Errorf("expected 0 checks, got %d", len(rm.checks))
		}
	})
}

func TestReadinessManager_CheckAll(t *testing.T) {
	t.Run("All checks healthy", func(t *testing.T) {
		rm := NewReadinessManager()
		rm.Register(NewAlwaysHealthyCheck("check1"))
		rm.Register(NewAlwaysHealthyCheck("check2"))

		results := rm.CheckAll(context.Background())

		if len(results) != 2 {
			t.Errorf("expected 2 results, got %d", len(results))
		}

		for name, result := range results {
			if result.Status != StatusHealthy {
				t.Errorf("expected check %s to be healthy, got %s", name, result.Status)
			}
		}
	})

	t.Run("Some checks unhealthy", func(t *testing.T) {
		rm := NewReadinessManager()
		rm.Register(NewAlwaysHealthyCheck("check1"))
		rm.Register(NewAlwaysUnhealthyCheck("check2"))

		results := rm.CheckAll(context.Background())

		if len(results) != 2 {
			t.Errorf("expected 2 results, got %d", len(results))
		}

		if results["check1"].Status != StatusHealthy {
			t.Error("expected check1 to be healthy")
		}

		if results["check2"].Status != StatusUnhealthy {
			t.Error("expected check2 to be unhealthy")
		}
	})

	t.Run("No checks registered", func(t *testing.T) {
		rm := NewReadinessManager()

		results := rm.CheckAll(context.Background())

		if len(results) != 0 {
			t.Errorf("expected 0 results, got %d", len(results))
		}
	})

	t.Run("Check timeout", func(t *testing.T) {
		rm := NewReadinessManager(WithTimeout(100 * time.Millisecond))

		slowCheck := NewCustomReadinessCheck("slow", func(ctx context.Context) error {
			select {
			case <-time.After(1 * time.Second):
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		}, "ok")

		rm.Register(slowCheck)

		results := rm.CheckAll(context.Background())

		if results["slow"].Status != StatusUnhealthy {
			t.Error("expected slow check to be unhealthy due to timeout")
		}
	})
}

func TestReadinessManager_IsReady(t *testing.T) {
	t.Run("All checks healthy returns true", func(t *testing.T) {
		rm := NewReadinessManager()
		rm.Register(NewAlwaysHealthyCheck("check1"))
		rm.Register(NewAlwaysHealthyCheck("check2"))

		if !rm.IsReady(context.Background()) {
			t.Error("expected IsReady to return true")
		}
	})

	t.Run("Any check unhealthy returns false", func(t *testing.T) {
		rm := NewReadinessManager()
		rm.Register(NewAlwaysHealthyCheck("check1"))
		rm.Register(NewAlwaysUnhealthyCheck("check2"))

		if rm.IsReady(context.Background()) {
			t.Error("expected IsReady to return false")
		}
	})

	t.Run("No checks returns true", func(t *testing.T) {
		rm := NewReadinessManager()

		if !rm.IsReady(context.Background()) {
			t.Error("expected IsReady to return true with no checks")
		}
	})
}

func TestReadinessManager_ParallelVsSequential(t *testing.T) {
	t.Run("Parallel execution", func(t *testing.T) {
		rm := NewReadinessManager() // parallel by default

		// Add checks that take 100ms each
		for i := range 5 {
			name := "check" + string(rune('0'+i))
			check := NewCustomReadinessCheck(name, func(ctx context.Context) error {
				time.Sleep(100 * time.Millisecond)
				return nil
			}, "ok")
			rm.Register(check)
		}

		start := time.Now()
		rm.CheckAll(context.Background())
		duration := time.Since(start)

		// Should take ~100ms if parallel, ~500ms if sequential
		if duration > 300*time.Millisecond {
			t.Errorf("parallel execution took too long: %v", duration)
		}
	})

	t.Run("Sequential execution", func(t *testing.T) {
		rm := NewReadinessManager(WithSequentialChecks())

		// Add checks that take 50ms each
		for i := range 3 {
			name := "check" + string(rune('0'+i))
			check := NewCustomReadinessCheck(name, func(ctx context.Context) error {
				time.Sleep(50 * time.Millisecond)
				return nil
			}, "ok")
			rm.Register(check)
		}

		start := time.Now()
		rm.CheckAll(context.Background())
		duration := time.Since(start)

		// Should take ~150ms if sequential
		if duration < 100*time.Millisecond {
			t.Errorf("sequential execution too fast: %v", duration)
		}
	})
}

func TestLivenessServer_ReadyzHandler(t *testing.T) {
	t.Run("All checks healthy returns 200", func(t *testing.T) {
		rm := NewReadinessManager()
		rm.Register(NewAlwaysHealthyCheck("check1"))
		rm.Register(NewAlwaysHealthyCheck("check2"))

		s := NewLivenessServer(rm)

		req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
		w := httptest.NewRecorder()

		s.ReadyzHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}

		var response map[string]any
		json.NewDecoder(resp.Body).Decode(&response)

		if response["status"] != "ready" {
			t.Errorf("expected status 'ready', got '%v'", response["status"])
		}
	})

	t.Run("Any check unhealthy returns 503", func(t *testing.T) {
		rm := NewReadinessManager()
		rm.Register(NewAlwaysHealthyCheck("check1"))
		rm.Register(NewAlwaysUnhealthyCheck("check2"))

		s := NewLivenessServer(rm)

		req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
		w := httptest.NewRecorder()

		s.ReadyzHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusServiceUnavailable {
			t.Errorf("expected status 503, got %d", resp.StatusCode)
		}

		var response map[string]any
		json.NewDecoder(resp.Body).Decode(&response)

		if response["status"] != "not ready" {
			t.Errorf("expected status 'not ready', got '%v'", response["status"])
		}

		if checks, ok := response["checks"].(map[string]any); ok {
			if len(checks) != 2 {
				t.Errorf("expected 2 checks, got %d", len(checks))
			}
			check1 := checks["check1"].(map[string]any)
			if check1["status"] != string(StatusHealthy) {
				t.Errorf("expected check1 to be healthy, got %s", check1["status"])
			}
			check2 := checks["check2"].(map[string]any)
			if check2["status"] != string(StatusUnhealthy) {
				t.Errorf("expected check2 to be unhealthy, got %s", check2["status"])
			}
		} else {
			t.Error("expected 'checks' field when unhealthy")
		}
	})

	t.Run("Verbose mode shows all checks when healthy", func(t *testing.T) {
		rm := NewReadinessManager()
		rm.Register(NewAlwaysHealthyCheck("check1"))

		s := NewLivenessServer(rm)

		req := httptest.NewRequest(http.MethodGet, "/readyz?verbose=true", nil)
		w := httptest.NewRecorder()

		s.ReadyzHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		var response map[string]any
		json.NewDecoder(resp.Body).Decode(&response)

		if checks, ok := response["checks"].(map[string]any); ok {
			if len(checks) != 1 {
				t.Errorf("expected 1 check, got %d", len(checks))
			}
			check1 := checks["check1"].(map[string]any)
			if check1["status"] != string(StatusHealthy) {
				t.Errorf("expected check1 to be healthy, got %s", check1["status"])
			}
		} else {
			t.Error("expected 'checks' field in verbose mode")
		}
	})

	t.Run("Shows checks when unhealthy even without verbose", func(t *testing.T) {
		rm := NewReadinessManager()
		rm.Register(NewAlwaysUnhealthyCheck("check1"))

		s := NewLivenessServer(rm)

		req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
		w := httptest.NewRecorder()

		s.ReadyzHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		var response map[string]any
		json.NewDecoder(resp.Body).Decode(&response)

		if checks, ok := response["checks"].(map[string]any); ok {
			if len(checks) != 1 {
				t.Errorf("expected 1 check, got %d", len(checks))
			}
			check1 := checks["check1"].(map[string]any)
			if check1["status"] != string(StatusUnhealthy) {
				t.Errorf("expected check1 to be unhealthy, got %s", check1["status"])
			}
		} else {
			t.Error("expected 'checks' field when unhealthy")
		}
	})

	t.Run("No checks registered returns ready", func(t *testing.T) {
		rm := NewReadinessManager()
		s := NewLivenessServer(rm)

		req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
		w := httptest.NewRecorder()

		s.ReadyzHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}
	})
}

func TestCustomReadinessCheck(t *testing.T) {
	t.Run("Successful check", func(t *testing.T) {
		check := NewCustomReadinessCheck("test", func(ctx context.Context) error {
			return nil
		}, "all good")

		result := check.Check(context.Background())

		if result.Status != StatusHealthy {
			t.Errorf("expected healthy status, got %s", result.Status)
		}
		if result.Message != "all good" {
			t.Errorf("expected message 'all good', got '%s'", result.Message)
		}
	})

	t.Run("Failed check", func(t *testing.T) {
		check := NewCustomReadinessCheck("test", func(ctx context.Context) error {
			return errors.New("something went wrong")
		}, "all good")

		result := check.Check(context.Background())

		if result.Status != StatusUnhealthy {
			t.Errorf("expected unhealthy status, got %s", result.Status)
		}
		if result.Error != "something went wrong" {
			t.Errorf("expected error 'something went wrong', got '%s'", result.Error)
		}
	})

	t.Run("Context cancellation", func(t *testing.T) {
		check := NewCustomReadinessCheck("test", func(ctx context.Context) error {
			select {
			case <-time.After(1 * time.Second):
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		}, "all good")

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		result := check.Check(ctx)

		if result.Status != StatusUnhealthy {
			t.Errorf("expected unhealthy status due to timeout, got %s", result.Status)
		}
	})
}

func TestLivenessServer_GetReadinessManager(t *testing.T) {
	t.Run("returns the correct manager", func(t *testing.T) {
		rm := NewReadinessManager()
		s := NewLivenessServer(rm)

		if s.GetReadinessManager() != rm {
			t.Error("expected the same readiness manager")
		}
	})
}

// Custom check implementations for testing

// AlwaysHealthyCheck is a check that always returns healthy
type AlwaysHealthyCheck struct {
	name string
}

func NewAlwaysHealthyCheck(name string) *AlwaysHealthyCheck {
	return &AlwaysHealthyCheck{name: name}
}

func (c *AlwaysHealthyCheck) Name() string {
	return c.name
}

func (c *AlwaysHealthyCheck) Check(ctx context.Context) CheckResult {
	return CheckResult{
		Name:   c.name,
		Status: StatusHealthy,
	}
}

// AlwaysUnhealthyCheck is a check that always returns unhealthy
type AlwaysUnhealthyCheck struct {
	name string
}

func NewAlwaysUnhealthyCheck(name string) *AlwaysUnhealthyCheck {
	return &AlwaysUnhealthyCheck{name: name}
}

func (c *AlwaysUnhealthyCheck) Name() string {
	return c.name
}

func (c *AlwaysUnhealthyCheck) Check(ctx context.Context) CheckResult {
	return CheckResult{
		Name:   c.name,
		Status: StatusUnhealthy,
		Error:  "i am always unhealthy",
	}
}

// CustomReadinessCheck is a check with a custom check function
type CustomReadinessCheck struct {
	name    string
	checkFn func(ctx context.Context) error
	message string
}

func NewCustomReadinessCheck(name string, checkFn func(ctx context.Context) error, message string) *CustomReadinessCheck {
	return &CustomReadinessCheck{
		name:    name,
		checkFn: checkFn,
		message: message,
	}
}

func (c *CustomReadinessCheck) Name() string {
	return c.name
}

func (c *CustomReadinessCheck) Check(ctx context.Context) CheckResult {
	if err := c.checkFn(ctx); err != nil {
		return CheckResult{
			Name:   c.name,
			Status: StatusUnhealthy,
			Error:  err.Error(),
		}
	}
	return CheckResult{
		Name:    c.name,
		Status:  StatusHealthy,
		Message: c.message,
	}
}
