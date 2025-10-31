package admin

import (
	"context"
	"fmt"
	"net/http"

	"github.com/htwr-aachen/backend/pkg/admin/config"
	"github.com/htwr-aachen/backend/pkg/admin/handlers"
	"github.com/htwr-aachen/backend/pkg/qa/db"
	"github.com/htwr-aachen/backend/pkg/session"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

type AdminService struct {
	DB       *db.DB
	Sessions *session.SessionSubsystem
}

func Init(ctx context.Context, conf *viper.Viper) (http.Handler, func(), error) {
	if conf == nil {
		log.Panic().Stack().Msg("nil conf given")
	}
	var err error
	var service AdminService

	cfg, err := config.LoadConfig(ctx, conf)
	if err != nil {
		return nil, nil, fmt.Errorf("configuring admin service: %w", err)
	}

	service.DB, err = db.New(ctx)
	if err != nil {
		return nil, nil, err
	}

	service.Sessions, err = session.New(ctx, conf, &session.SessionUsageConfig{
		AuthLoginURL: "/admin/login",
	})
	if err != nil {
		return nil, nil, err
	}

	h := handlers.New(ctx, cfg, service.DB, service.Sessions)

	closer := func() {}
	return h, closer, nil
}
