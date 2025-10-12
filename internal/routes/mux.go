// Package routes
package routes

import (
	"net/http"

	"github.com/ntentasd/nostradamus-api/pkg/utils"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func NewMux(app *App) http.Handler {
	mux := http.NewServeMux()

	// health check
	mux.HandleFunc("/healthz", healthHandler)

	// metrics
	mux.Handle("/metrics", promhttp.Handler())

	// get 5 latest values
	mux.HandleFunc("/latest", app.latestHandler)

	// get fields & sensors
	mux.HandleFunc("/fields", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			app.fieldsHandler(w, r)
		case http.MethodPost:
			app.registerFieldHandler(w, r)
		default:
			utils.ReplyMethodNotAllowed(w)
		}
	})
	mux.HandleFunc("/sensors", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			app.sensorsHandler(w, r)
		case http.MethodPost:
			app.registerSensorHandler(w, r)
		default:
			utils.ReplyMethodNotAllowed(w)
		}
	})

	// arroyo command routes
	mux.HandleFunc("/jobs", app.ListJobs)
	mux.HandleFunc("/jobs/{id}", app.GetJob)
	mux.HandleFunc("/jobs/run", app.CreatePipeline)
	mux.HandleFunc("/pipelines", app.ListPipelines)

	return utils.WithCORS(mux)
}
