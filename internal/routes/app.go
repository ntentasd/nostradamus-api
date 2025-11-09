package routes

import (
	"github.com/ntentasd/nostradamus-api/internal/arroyo"
	"github.com/ntentasd/nostradamus-api/internal/cache"
	"github.com/ntentasd/nostradamus-api/internal/db"
	"github.com/ntentasd/nostradamus-api/internal/emqx"
	"github.com/rs/zerolog"
)

type App struct {
	Store *db.DB
	Cache cache.Cache
	*arroyo.ArroyoClient
	*emqx.EmqxClient
	logger zerolog.Logger
}

func New(store *db.DB, cache cache.Cache, ac *arroyo.ArroyoClient, ec *emqx.EmqxClient, logger zerolog.Logger) *App {
	return &App{
		store,
		cache,
		ac,
		ec,
		logger,
	}
}
