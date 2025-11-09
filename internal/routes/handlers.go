package routes

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gocql/gocql"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	"github.com/ntentasd/nostradamus-api/internal/db"
	"github.com/ntentasd/nostradamus-api/internal/metrics"
	"github.com/ntentasd/nostradamus-api/pkg/types"
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
	res, err := app.Cache.FetchLast(
		fmt.Sprintf(
			"%s:%s:%s",
			"sensor",
			"550e8400-e29b-41d4-a716-446655440000",
			today.Format("2006-01-02"),
		),
		5)
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
		res, err = app.Store.GetLast5Values(
			"550e8400-e29b-41d4-a716-446655440000",
			today.Format("2006-01-02"),
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
		// TODO: Create pipelined function
		for _, entry := range res {
			app.Cache.Store(
				fmt.Sprintf(
					"%s:%s:%s",
					"sensor",
					"550e8400-e29b-41d4-a716-446655440000",
					today,
				),
				entry,
			)
		}
	}

	utils.ReplyJSON(w, http.StatusOK, utils.Body{
		"data": res,
	})
}

func (app *App) aggregateHandler(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer("nostradamus-api").Start(r.Context(), "HTTP /aggregate")
	defer span.End()

	start := time.Now()

	defer func() {
		metrics.HttpRequestLatencySeconds.WithLabelValues("GET").Observe(time.Since(start).Seconds())
	}()

	sensorID := r.URL.Query().Get("sensor_id")
	sensorType := r.URL.Query().Get("sensor_type")
	windowStr := r.URL.Query().Get("window")
	if sensorID == "" || sensorType == "" || windowStr == "" {
		utils.ReplyBadRequest(w, "missing query params")
		return
	}

	sType, err := strconv.Atoi(sensorType)
	if err != nil || sType < 0 || sType > 2 {
		utils.ReplyBadRequest(w, "invalid sensor type")
		return
	}

	dur, err := time.ParseDuration(windowStr)
	if err != nil {
		utils.ReplyBadRequest(w, "invalid window")
		return
	}

	now := time.Now().UTC()
	today := now.Format("2006-01-02")
	cacheKey := fmt.Sprintf("agg:%s:%s:%s", sensorID, today, windowStr)

	cached, err := app.Cache.FetchAggregate(ctx, cacheKey)
	if err == nil && cached != nil {
		var agg types.Aggregate
		if err = json.Unmarshal(cached, &agg); err == nil {
			utils.ReplyJSON(w, http.StatusOK, utils.Body{
				"data": agg,
			})
			span.SetStatus(codes.Ok, "")
			return
		}

		span.SetAttributes(attribute.String("cache.result", "corrupt"))
		span.RecordError(err)
	}

	readings, err := app.Store.GetReadings(ctx, sensorID, sType, now.Add(-dur), now)
	if err != nil {
		utils.ReplyInternalServerError(w, err.Error())
		return
	}

	if len(readings) == 0 {
		utils.ReplyNotFound(w, "no readings found")
		return
	}

	sum, min, max := 0.0, readings[0], readings[0]
	for _, v := range readings {
		sum += v
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}

	agg := types.Aggregate{
		Avg:       sum / float64(len(readings)),
		Min:       min,
		Max:       max,
		Count:     len(readings),
		Timestamp: now,
	}

	_ = app.Cache.StoreAggregate(ctx, cacheKey, agg, time.Minute*5)

	utils.ReplyJSON(w, http.StatusOK, utils.Body{
		"data": agg,
	})
	span.SetStatus(codes.Ok, "")
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
		http.Error(
			w,
			fmt.Sprintf("db error: %v", err),
			http.StatusInternalServerError,
		)
		return
	}

	utils.ReplyJSON(w, http.StatusOK, utils.Body{
		"data": fields,
	})
}

func (app *App) getFieldByIDHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.ReplyMethodNotAllowed(w)
		return
	}

	fieldIDstr := r.URL.Query().Get("field_id")
	if fieldIDstr == "" {
		utils.ReplyJSON(w, http.StatusBadRequest, utils.Body{
			"error": "missing field_id",
		})
		return
	}

	fieldID, err := uuid.Parse(fieldIDstr)
	if err != nil {
		utils.ReplyJSON(w, http.StatusBadRequest, utils.Body{
			"error": "invalid field_id",
		})
		return
	}

	field, err := app.Store.GetFieldByID(fieldID)
	if err != nil {
		utils.ReplyJSON(w, http.StatusInternalServerError, utils.Body{
			"error": err.Error(),
		})
		return
	}

	if field == nil {
		utils.ReplyJSON(w, http.StatusNotFound, utils.Body{
			"data": nil,
		})
		return
	}

	utils.ReplyJSON(w, http.StatusOK, utils.Body{
		"data": field,
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
		http.Error(
			w,
			fmt.Sprintf("db error: %v", err),
			http.StatusInternalServerError,
		)
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

	sensor, err := app.Store.RegisterSensor(
		fieldUUID,
		req.SensorName,
		req.SensorType,
	)
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

	username := fmt.Sprintf(
		"%s_%s",
		sensor.SensorID.String()[:8],
		req.SensorName,
	)
	password := uuid.NewString()[:12]

	if _, err = app.CreateUser(username, password, false); err != nil {
		app.logger.Warn().Err(err).Str("sensor", username).Msg("failed to create EMQX user")
	} else {
		app.logger.Info().Str("username", username).Str("sensor_name", sensor.SensorName).Msg("EMQX user created successfully")
	}

	err = app.Store.StoreSensorCredentials(
		gocql.UUID(fieldUUID),
		gocql.UUID(sensor.SensorID),
		username,
		password,
	)
	if err != nil {
		app.logger.Warn().Err(err).Str("sensor_id", sensor.SensorID.String()).Msg("failed to store MQTT credentials")
	}

	utils.ReplyJSON(w, http.StatusCreated, utils.Body{
		"data": map[string]any{
			"sensor":    sensor,
			"mqtt_user": username,
			"mqtt_pass": password,
		},
	})
}

func (app *App) getSensorCredentialsHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodGet {
		utils.ReplyMethodNotAllowed(w)
		return
	}

	sensorIDStr := r.URL.Query().Get("sensor_id")
	if sensorIDStr == "" {
		utils.ReplyJSON(w, http.StatusBadRequest, utils.Body{
			"error": "missing sensor_id",
		})
		return
	}

	sensorID, err := uuid.Parse(sensorIDStr)
	if err != nil {
		utils.ReplyJSON(w, http.StatusBadRequest, utils.Body{
			"error": "invalid sensor_id",
		})
		return
	}

	creds, err := app.Store.GetSensorCredentials(sensorID)
	if err != nil {
		if errors.Is(err, db.ErrSensorNotFound) {
			utils.ReplyJSON(w, http.StatusNotFound, utils.Body{
				"error": "sensor not found",
			})
			return
		}

		utils.ReplyJSON(w, http.StatusInternalServerError, utils.Body{
			"error": err.Error(),
		})
		return
	}

	utils.ReplyJSON(w, http.StatusOK, utils.Body{
		"data": creds,
	})
}
