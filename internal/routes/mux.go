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
	store  *db.DB
	cache  *cache.DB
	client *ArroyoClient
}

func NewMux(store *db.DB, cache *cache.DB, arroyoURL string) http.Handler {
	mux := http.NewServeMux()

	client := New(arroyoURL)

	app := App{
		store,
		cache,
		client,
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

	// arroyo command routes
	mux.HandleFunc("/jobs", app.listJobs)
	mux.HandleFunc("/jobs/run", app.createPipeline)

	return utils.WithCORS(mux)
}
