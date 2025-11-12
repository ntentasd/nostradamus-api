package routes

import (
	"context"
	"time"

	"github.com/ntentasd/nostradamus-api/internal/arroyo"
	"github.com/ntentasd/nostradamus-api/internal/cache"
	"github.com/ntentasd/nostradamus-api/internal/db"
	"github.com/ntentasd/nostradamus-api/internal/emqx"
	"github.com/rs/zerolog"
)

type Config struct {
	driver string
}

type App struct {
	Store *db.DB
	Cache cache.Cache
	*arroyo.ArroyoClient
	*emqx.EmqxClient
	logger zerolog.Logger
	config *Config
}

func NewConfig(driver string) *Config {
	return &Config{
		driver,
	}
}

func New(store *db.DB, cache cache.Cache, ac *arroyo.ArroyoClient, ec *emqx.EmqxClient, logger zerolog.Logger, config *Config) *App {
	return &App{
		store,
		cache,
		ac,
		ec,
		logger,
		config,
	}
}

func (app *App) WarmUp() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_ = app.Store.Meta.Query(`SELECT now() FROM system.local`).WithContext(ctx).Exec()
	_ = app.Cache.Ping(ctx)
}
