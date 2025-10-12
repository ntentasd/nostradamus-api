package routes

import (
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/ntentasd/nostradamus-api/pkg/utils"
)

func healthHandler(w http.ResponseWriter, r *http.Request) {
	utils.ReplyJSON(w, http.StatusOK, utils.Body{
		"state": "healthy",
	})
}

func (app *App) latestHandler(w http.ResponseWriter, r *http.Request) {
	year, month, day := time.Now().Date()
	today := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
	res, err := app.Cache.FetchLast5(
		fmt.Sprintf("%s:%s:%s", "sensor", "550e8400-e29b-41d4-a716-446655440000", today.Format("2006-01-02")),
	)
	if err != nil {
		utils.ReplyJSON(
			w,
			http.StatusInternalServerError,
			map[string]any{
				"error": err.Error(),
			},
		)
		return
	}

	// Less than 5, cache is stale
	if len(res) < 5 {
		res, err = app.Store.GetLast5Values("550e8400-e29b-41d4-a716-446655440000", today.Format("2006-01-02"))
		if err != nil {
			utils.ReplyJSON(
				w,
				http.StatusInternalServerError,
				map[string]any{
					"error": err.Error(),
				},
			)
			return
		}
		// Create pipelined function
		for _, entry := range res {
			app.Cache.Store(
				fmt.Sprintf("%s:%s:%s", "sensor", "550e8400-e29b-41d4-a716-446655440000", today),
				entry,
			)
		}
	}

	utils.ReplyJSON(w, http.StatusOK, utils.Body{
		"data": res,
	})
}

func (app *App) fieldsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.ReplyMethodNotAllowed(w)
		return
	}

	userIDstr := r.URL.Query().Get("user_id")
	if userIDstr == "" {
		http.Error(w, "missing user_id", http.StatusBadRequest)
		return
	}

	userID, err := uuid.Parse(userIDstr)
	if err != nil {
		http.Error(w, "invalid user_id", http.StatusBadRequest)
		return
	}

	fields, err := app.Store.GetFieldsByUserID(userID)
	if err != nil {
		http.Error(w, fmt.Sprintf("db error: %v", err), http.StatusInternalServerError)
		return
	}

	utils.ReplyJSON(w, http.StatusOK, utils.Body{
		"data": fields,
	})
}

func (app *App) sensorsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.ReplyMethodNotAllowed(w)
		return
	}

	fieldIDstr := r.URL.Query().Get("field_id")
	if fieldIDstr == "" {
		http.Error(w, "missing field_id", http.StatusBadRequest)
		return
	}

	fieldID, err := uuid.Parse(fieldIDstr)
	if err != nil {
		http.Error(w, "invalid field_id", http.StatusBadRequest)
		return
	}

	sensors, field, err := app.Store.GetSensorsByFieldID(fieldID)
	if err != nil {
		http.Error(w, fmt.Sprintf("db error: %v", err), http.StatusInternalServerError)
		return
	}

	utils.ReplyJSON(w, http.StatusOK, utils.Body{
		"data":  sensors,
		"field": *field,
	})
}
