package routes

import (
	"github.com/ntentasd/nostradamus-api/internal/arroyo"
	"github.com/ntentasd/nostradamus-api/internal/cache"
	"github.com/ntentasd/nostradamus-api/internal/db"
)

type App struct {
	Store *db.DB
	Cache *cache.DB
	*arroyo.ArroyoClient
}

func New(store *db.DB, cache *cache.DB, ac *arroyo.ArroyoClient) *App {
	return &App{
		store,
		cache,
		ac,
	}
}
