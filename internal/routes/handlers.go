package routes

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/ntentasd/nostradamus-api/internal/db"
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

func (app *App) registerFieldHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.ReplyMethodNotAllowed(w)
		return
	}

	var req struct {
		UserID    string `json:"user_id"`
		FieldName string `json:"field_name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.ReplyJSON(w, http.StatusBadRequest, utils.Body{
			"error": "invalid request body",
		})
		return
	}

	userUUID, err := uuid.Parse(req.UserID)
	if err != nil {
		utils.ReplyJSON(w, http.StatusBadRequest, utils.Body{
			"error": "invalid user_id",
		})
		return
	}

	field, err := app.Store.RegisterField(userUUID, req.FieldName)
	if err != nil {
		var dupErr *db.FieldAlreadyExistsError
		if errors.As(err, &dupErr) {
			utils.ReplyJSON(w, http.StatusBadRequest, utils.Body{
				"error": dupErr.Error(),
			})
			return
		}

		utils.ReplyJSON(w, http.StatusInternalServerError, utils.Body{
			"error": err.Error(),
		})
		return
	}

	utils.ReplyJSON(w, http.StatusCreated, utils.Body{
		"data": field,
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

func (app *App) registerSensorHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.ReplyMethodNotAllowed(w)
		return
	}

	var req struct {
		FieldID    string `json:"field_id"`
		SensorName string `json:"sensor_name"`
		SensorType int    `json:"sensor_type"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.ReplyJSON(w, http.StatusBadRequest, utils.Body{
			"error": "invalid request body",
		})
		return
	}

	fieldUUID, err := uuid.Parse(req.FieldID)
	if err != nil {
		utils.ReplyJSON(w, http.StatusBadRequest, utils.Body{
			"error": "invalid field UUID",
		})
		return
	}

	// Register sensor via DB layer
	sensor, err := app.Store.RegisterSensor(fieldUUID, req.SensorName, req.SensorType)
	if err != nil {
		var dupErr *db.SensorAlreadyExistsError
		if errors.As(err, &dupErr) {
			utils.ReplyJSON(w, http.StatusBadRequest, utils.Body{
				"error": dupErr.Error(),
			})
			return
		}
		utils.ReplyJSON(w, http.StatusInternalServerError, utils.Body{
			"error": err.Error(),
		})
		return
	}

	utils.ReplyJSON(w, http.StatusCreated, utils.Body{
		"data": sensor,
	})
}
