package liveness

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// CheckStatus represents the status of a readiness check
type CheckStatus string

const (
	StatusHealthy   CheckStatus = "healthy"
	StatusUnhealthy CheckStatus = "unhealthy"
	StatusUnknown   CheckStatus = "unknown"
)

// CheckResult represents the result of a single readiness check
type CheckResult struct {
	Name    string      `json:"name"`
	Status  CheckStatus `json:"status"`
	Message string      `json:"message,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// ReadinessCheck is the interface that subsystems must implement
type ReadinessCheck interface {
	// Name returns the name of the check
	Name() string

	// Check performs the readiness check and returns the result
	// The context can be used for timeout and cancellation
	Check(ctx context.Context) CheckResult
}

// ReadinessManager manages multiple readiness checks
type ReadinessManager struct {
	checks   map[string]ReadinessCheck
	mu       sync.RWMutex
	timeout  time.Duration
	parallel bool // if true, run checks in parallel
}

// NewReadinessManager creates a new ReadinessManager
func NewReadinessManager(opts ...ManagerOption) *ReadinessManager {
	rm := &ReadinessManager{
		checks:   make(map[string]ReadinessCheck),
		timeout:  5 * time.Second, // default timeout
		parallel: true,            // run checks in parallel by default
	}

	for _, opt := range opts {
		opt(rm)
	}

	return rm
}

// ManagerOption is a functional option for configuring ReadinessManager
type ManagerOption func(*ReadinessManager)

// WithTimeout sets the timeout for readiness checks
func WithTimeout(timeout time.Duration) ManagerOption {
	return func(rm *ReadinessManager) {
		rm.timeout = timeout
	}
}

// WithSequentialChecks configures checks to run sequentially instead of in parallel
func WithSequentialChecks() ManagerOption {
	return func(rm *ReadinessManager) {
		rm.parallel = false
	}
}

// Register adds a readiness check to the manager
func (rm *ReadinessManager) Register(check ReadinessCheck) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	name := check.Name()
	if _, exists := rm.checks[name]; exists {
		log.Warn().
			Str("check_name", name).
			Msg("readiness check already registered, replacing")
	}

	rm.checks[name] = check
	log.Info().
		Str("check_name", name).
		Msg("registered readiness check")
}

// Unregister removes a readiness check from the manager
func (rm *ReadinessManager) Unregister(name string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	delete(rm.checks, name)
	log.Info().
		Str("check_name", name).
		Msg("unregistered readiness check")
}

// Close removes all readiness check from the manager
func (rm *ReadinessManager) Close() {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	rm.checks = make(map[string]ReadinessCheck)
	log.Info().Msg("closed readiness checks")
}

// CheckAll runs all registered readiness checks
func (rm *ReadinessManager) CheckAll(ctx context.Context) map[string]CheckResult {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	if len(rm.checks) == 0 {
		return make(map[string]CheckResult)
	}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(ctx, rm.timeout)
	defer cancel()

	if rm.parallel {
		return rm.checkParallel(ctx)
	}
	return rm.checkSequential(ctx)
}

// checkParallel runs all checks in parallel
func (rm *ReadinessManager) checkParallel(ctx context.Context) map[string]CheckResult {
	results := make(map[string]CheckResult)
	resultsChan := make(chan CheckResult, len(rm.checks))
	var wg sync.WaitGroup

	for _, check := range rm.checks {
		wg.Add(1)
		go func(c ReadinessCheck) {
			defer wg.Done()

			result := c.Check(ctx)
			resultsChan <- result
		}(check)
	}

	// Close the channel when all checks are done
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Collect results
	for result := range resultsChan {
		results[result.Name] = result
	}

	return results
}

// checkSequential runs all checks sequentially
func (rm *ReadinessManager) checkSequential(ctx context.Context) map[string]CheckResult {
	results := make(map[string]CheckResult)

	for _, check := range rm.checks {
		result := check.Check(ctx)
		results[result.Name] = result
	}

	return results
}

// IsReady returns true if all checks are healthy
func (rm *ReadinessManager) IsReady(ctx context.Context) bool {
	results := rm.CheckAll(ctx)

	for _, result := range results {
		if result.Status != StatusHealthy {
			return false
		}
	}

	return true
}

// ReadyzHandler returns 200 OK if the server is ready to accept traffic
func (s *Server) ReadyzHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	results := s.readinessManager.CheckAll(ctx)

	// Check if verbose output is requested
	verbose := r.URL.Query().Get("verbose") == "true"

	// Determine overall status
	allHealthy := true
	for _, result := range results {
		if result.Status != StatusHealthy {
			allHealthy = false
			break
		}
	}

	w.Header().Set("Content-Type", "application/json")

	if allHealthy {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	// Build response
	response := make(map[string]any)
	if allHealthy {
		response["status"] = "ready"
	} else {
		response["status"] = "not ready"
	}

	if verbose || !allHealthy {
		response["checks"] = results
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Err(err).Msg("writing readiness response")
	}
}

// GetReadinessManager returns the readiness manager (useful for registering checks)
func (s *Server) GetReadinessManager() *ReadinessManager {
	return s.readinessManager
}
