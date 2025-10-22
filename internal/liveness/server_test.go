package liveness

import (
	"testing"
)

func TestNewLivenessServer(t *testing.T) {
	t.Run("with nil readiness manager", func(t *testing.T) {
		s := NewLivenessServer(nil)
		if s.readinessManager == nil {
			t.Error("readinessManager should be initialized even if nil is passed")
		}
	})

	t.Run("with existing readiness manager", func(t *testing.T) {
		rm := NewReadinessManager()
		s := NewLivenessServer(rm)
		if s.readinessManager != rm {
			t.Error("should use the passed readinessManager")
		}
	})
}