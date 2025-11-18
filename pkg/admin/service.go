package admin

import (
	"context"
	"net/http"

	"github.com/htwr-aachen/backend/internal/configurator"
	"github.com/htwr-aachen/backend/pkg/admin/handlers"
	"github.com/htwr-aachen/backend/pkg/config"
	"github.com/htwr-aachen/backend/pkg/qa/db"
	"github.com/htwr-aachen/backend/pkg/session"
	"github.com/rs/zerolog/log"
)

type AdminService struct {
	DB       *db.DB
	Sessions *session.SessionSubsystem
}

func Init(ctx context.Context) (http.Handler, func(), error) {
	gcfg, ok := configurator.FromContext(ctx)
	if !ok {
		log.Panic().Stack().Msg("no configuration context given")
	}
	var err error
	var service AdminService

	service.DB, err = db.New(ctx)
	if err != nil {
		return nil, nil, err
	}

	service.Sessions, err = session.New(ctx, &config.SessionUsageConfig{
		AuthLoginURL: "/admin/login",
	})
	if err != nil {
		return nil, nil, err
	}

	h := handlers.New(ctx, gcfg, service.DB, service.Sessions)

	closer := func() {}
	return h, closer, nil
}
