package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/htwr-aachen/backend/pkg/panikzettel/models"
	"github.com/rs/zerolog/log"
)

type PanikzettelDB interface {
	GetPanikzettelMeta(ctx context.Context) ([]models.PanikzettelMeta, error)
	GetPanikzettel(ctx context.Context, name string) (*models.Panikzettel, error)
}

type Panikzettel struct {
	db PanikzettelDB
}

func NewPanikzettel(db PanikzettelDB) *Panikzettel {
	return &Panikzettel{
		db: db,
	}
}

// GetPanikzettelMeta handles the GET /panikzettel/ request and returns a json array of panikzettel metas
func (h *Panikzettel) GetPanikzettelMeta(w http.ResponseWriter, r *http.Request) {
	panikzettel, err := h.db.GetPanikzettelMeta(r.Context())
	log.Debug().Msg("Looking up panikzettel cache")
	if err != nil {
		log.Err(err).Msg("Could not get Panikzettel metas")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	if err = encoder.Encode(panikzettel); err != nil {
		log.Err(err).Msg("Could not encode Panizettel Metadata")
	}
}

func (h *Panikzettel) GetPanikzettel(w http.ResponseWriter, r *http.Request) {
	filename := r.PathValue("filename")
	log.Trace().Str("filename", filename).Msg("Downloading Panikzettel")

	panikzettel, err := h.db.GetPanikzettel(r.Context(), filename)
	if err != nil {
		var notFoundErr *models.PanikzettelNotFoundError
		if errors.As(err, &notFoundErr) {
			log.Warn().Str("filename", filename).Msg("Unknown Panikzettel Requested")
			http.Error(w, "Not Found", http.StatusNotFound)
		} else {
			log.Err(err).Str("filename", filename).Msg("Could not get panikzettel")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", panikzettel.ContentType)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", panikzettel.Size))
	w.Header().Set("Last-Modified", panikzettel.LastModified.Format(http.TimeFormat))

	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))

	_, err = w.Write(panikzettel.Content)
	if err != nil {
		log.Err(err).Str("filename", filename).Msg("Failed to write panikzettel content")
	}
}
