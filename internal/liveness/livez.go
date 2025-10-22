package liveness

import (
	"net/http"

	"github.com/rs/zerolog/log"
)

// livezHandler returns 200 OK if the server is alive (process is running)
func (s *Server) LivezHandler(w http.ResponseWriter, _r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, err := w.Write([]byte(`{"status":"ok"}`))
	if err != nil {
		log.Err(err).Msg("writing liveness response")
	}
}
