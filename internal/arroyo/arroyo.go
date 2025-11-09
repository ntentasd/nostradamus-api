package arroyo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/ntentasd/nostradamus-api/pkg/types"
	"github.com/rs/zerolog"
)

type ArroyoClient struct {
	BaseURL string
	Client  *http.Client
	logger  zerolog.Logger
}

func New(baseURL string, logger zerolog.Logger) *ArroyoClient {
	return &ArroyoClient{
		BaseURL: baseURL,
		Client: &http.Client{
			Timeout: 10 * time.Second,
		},
		logger: logger,
	}
}

const JSONSchema = `{
  "type": "object",
  "properties": {
    "sensor_id": { "type": "string", "format": "uuid" },
    "bucket_date": { "type": "string", "format": "date" },
    "timestamp": { "type": "string", "format": "date-time" },
    "value": { "type": "number" }
  },
  "required": ["sensor_id", "bucket_date", "timestamp", "value"]
}`

func (ac *ArroyoClient) Get(path string) (*http.Response, error) {
	url := fmt.Sprintf("http://%s%s", ac.BaseURL, path)
	return ac.Client.Get(url)
}

func (ac *ArroyoClient) Post(path string, body any) (*http.Response, error) {
	url := fmt.Sprintf("http://%s%s", ac.BaseURL, path)

	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewBuffer(b)
	}

	req, err := http.NewRequest(http.MethodPost, url, reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	return ac.Client.Do(req)
}

func (ac *ArroyoClient) Delete(path string, body any) (*http.Response, error) {
	url := fmt.Sprintf("http://%s%s", ac.BaseURL, path)

	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewBuffer(b)
	}

	req, err := http.NewRequest(http.MethodDelete, url, reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	return ac.Client.Do(req)
}

func (ac *ArroyoClient) RestartPipeline(id string) error {
	// fetch pipeline definition
	resp, err := ac.Get(fmt.Sprintf("/api/v1/pipelines/%s", id))
	if err != nil {
		ac.logger.Error().Err(err).Str("pipeline_id", id).Msg("failed to fetch pipeline")
		return fmt.Errorf("failed to fetch pipeline %s: %w", id, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		ac.logger.Error().Str("status", resp.Status).Int("status_code", resp.StatusCode).Str("pipeline", string(body)).Msg("unexpected status while fetching pipeline")
		return fmt.Errorf("unexpected status %d while fetching pipeline: %s",
			resp.StatusCode, string(body))
	}

	var existing types.Pipeline
	if err := json.NewDecoder(resp.Body).Decode(&existing); err != nil {
		ac.logger.Error().Err(err).Msg("failed to decode pipeline")
		return fmt.Errorf("failed to decode pipeline: %w", err)
	}

	// delete old pipeline
	delResp, err := ac.Delete(fmt.Sprintf("/api/v1/pipelines/%s", id), nil)
	if err != nil {
		ac.logger.Error().Err(err).Str("pipeline_id", id).Msg("failed to delete pipeline")
		return fmt.Errorf("failed to delete pipeline %s: %w", id, err)
	}
	defer delResp.Body.Close()

	if delResp.StatusCode != http.StatusOK && delResp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(delResp.Body)
		ac.logger.Error().Err(err).Str("status", delResp.Status).Int("status_code", delResp.StatusCode).Str("pipeline", string(body)).Msg("unexpected status while reading pipeline")
		return fmt.Errorf("unexpected status %d while deleting pipeline: %s",
			delResp.StatusCode, string(body))
	}

	time.Sleep(500 * time.Millisecond)

	parallelism := 1
	if existing.Parallelism != nil {
		parallelism = *existing.Parallelism
	}

	// recreate payload
	var payload = struct {
		Name                     string `json:"name"`
		Query                    string `json:"query"`
		Parallelism              int    `json:"parallelism"`
		CheckpointIntervalMicros int64  `json:"checkpointIntervalMicros"`
		UDFs                     []any  `json:"udfs"`
	}{
		Name:                     existing.Name,
		Query:                    existing.Query,
		Parallelism:              parallelism,
		CheckpointIntervalMicros: 60000000,
		UDFs:                     []any{},
	}

	// recreate
	recreateResp, err := ac.Post("/api/v1/pipelines", payload)
	if err != nil {
		ac.logger.Error().Err(err).Str("pipeline_id", id).Msg("failed to create pipeline")
		return fmt.Errorf("failed to recreate pipeline %s: %w", id, err)
	}
	defer recreateResp.Body.Close()

	if recreateResp.StatusCode != http.StatusOK && recreateResp.StatusCode != http.StatusCreated {
		msg, _ := io.ReadAll(recreateResp.Body)
		ac.logger.Error().Err(err).Str("status", recreateResp.Status).Int("status_code", recreateResp.StatusCode).Str("pipeline", string(msg)).Msg("unexpected status while recreating pipeline")
		return fmt.Errorf("unexpected status %d while recreating pipeline: %s",
			recreateResp.StatusCode, string(msg))
	}

	return nil
}
