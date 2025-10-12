package routes

import (
	"github.com/ntentasd/nostradamus-api/internal/arroyo"
	"github.com/ntentasd/nostradamus-api/internal/cache"
	"github.com/ntentasd/nostradamus-api/internal/db"
	"github.com/ntentasd/nostradamus-api/internal/emqx"
)

type App struct {
	Store *db.DB
	Cache *cache.DB
	*arroyo.ArroyoClient
	*emqx.EmqxClient
}

func New(store *db.DB, cache *cache.DB, ac *arroyo.ArroyoClient, ec *emqx.EmqxClient) *App {
	return &App{
		store,
		cache,
		ac,
		ec,
	}
}
