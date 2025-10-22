package liveness

// Server handles liveness and readiness probes
type Server struct {
	readinessManager *ReadinessManager
}

// NewLivenessServer creates a new LivenessServer
func NewLivenessServer(readinessManager *ReadinessManager) *Server {
	if readinessManager == nil {
		readinessManager = NewReadinessManager()
	}

	return &Server{
		readinessManager: readinessManager,
	}
}

func (s *Server) Close() {
	s.readinessManager.Close()
}
