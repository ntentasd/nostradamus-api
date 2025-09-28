// Package routes
package routes

import (
	"net/http"

	"github.com/ntentasd/nostradamus-api/internal/cache"
	"github.com/ntentasd/nostradamus-api/internal/db"
	"github.com/ntentasd/nostradamus-api/pkg/utils"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type App struct {
	store *db.DB
	cache *cache.DB
}

func NewMux(store *db.DB, cache *cache.DB) http.Handler {
	mux := http.NewServeMux()

	app := App{
		store,
		cache,
	}

	// health check
	mux.HandleFunc("/healthz", healthHandler)

	// metrics
	mux.Handle("/metrics", promhttp.Handler())

	// get 5 latest values
	mux.HandleFunc("/latest", app.latestHandler)

	// get my fields
	mux.HandleFunc("/fields", app.fieldsHandler)
	mux.HandleFunc("/sensors", app.sensorsHandler)

	return utils.WithCORS(mux)
}
