package server

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
)

func (s *Server) setupPublicRouter() http.Handler {
	r := http.NewServeMux()
	if s.config.PanikzettelEnabled {
		r.Handle("/api/panikzettel/", http.StripPrefix("/api/panikzettel", s.services.Panikzettel))
	}
	if s.config.QAEnabled {
		log.Debug().Msg("Enabling QA routes")
		r.Handle("/api/qa/", http.StripPrefix("/api/qa", s.services.QA))
	}
	return r
}

func (s *Server) setupAdminRouter() *http.ServeMux {
	r := http.NewServeMux()
	if s.config.AdminEnabled {
		r.Handle("/", s.services.Admin)
	}
	return r
}

func (s *Server) setupMetricsRouter() *mux.Router {
	r := mux.NewRouter()
	r.Handle("/metrics", promhttp.Handler())
	r.HandleFunc("/livez", s.services.Liveness.LivezHandler)
	r.HandleFunc("/readyz", s.services.Liveness.ReadyzHandler)
	return r
}
