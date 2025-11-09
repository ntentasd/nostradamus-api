package arroyo

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/ntentasd/nostradamus-api/pkg/types"
	"github.com/ntentasd/nostradamus-api/pkg/utils"
)

func (ac *ArroyoClient) ListPipelinesInternal() ([]PipelineResponse, error) {
	resp, err := ac.Get("/api/v1/pipelines")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var wrapper struct {
		Data []PipelineResponse `json:"data"`
	}

	if err = json.NewDecoder(resp.Body).Decode(&wrapper); err != nil {
		return nil, err
	}

	return wrapper.Data, nil
}

func (ac *ArroyoClient) ListPipelines(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.ReplyMethodNotAllowed(w)
		return
	}

	defer r.Body.Close()

	pipelines, err := ac.ListPipelinesInternal()
	if err != nil {
		utils.ReplyJSON(w, http.StatusInternalServerError, utils.Body{
			"error": err.Error(),
		})
	}

	utils.ReplyJSON(w, http.StatusOK, utils.Body{
		"data": pipelines,
	})
}

func (ac *ArroyoClient) CreatePipeline(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.ReplyMethodNotAllowed(w)
		return
	}

	defer r.Body.Close()

	var body struct {
		Name  string `json:"name"`
		Field string `json:"field"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		ac.logger.Warn().Err(err).Msg("invalid request parameters")
		utils.ReplyJSON(w, http.StatusBadRequest, utils.Body{
			"error": err.Error(),
		})
		return
	}

	// TODO: standardize but make dynamic
	query := `
INSERT INTO temperatures_docker
SELECT
    *
FROM
    kafka2
`

	ptr := new(int)
	*ptr = 3
	req := types.Pipeline{
		Name:        body.Name,
		Query:       query,
		Parallelism: ptr,
	}

	resp, err := ac.Post("/api/v1/pipelines", req)
	if err != nil {
		ac.logger.Error().Err(err).Msg("failed to reach Arroyo API")
		utils.ReplyJSON(w, http.StatusBadGateway, utils.Body{
			"error": "failed to reach Arroyo API: " + err.Error(),
		})
		return
	}
	defer resp.Body.Close()

	var wrapper struct {
		Data []types.Pipeline `json:"data"`
	}

	if err = json.NewDecoder(resp.Body).Decode(&wrapper); err != nil {
		ac.logger.Error().Err(err).Msg("failed to decode Arroyo response")
		utils.ReplyJSON(w, http.StatusInternalServerError, utils.Body{
			"error": "failed to decode Arroyo response: " + err.Error(),
		})
		return
	}

	utils.ReplyJSON(w, http.StatusOK, utils.Body{
		"data": wrapper.Data,
	})
}

// TODO: merge with upper method, clean up
func (ac *ArroyoClient) CreatePipelineInternal(name, query string, parallelism int) (*PipelineResponse, error) {
	payload := map[string]any{
		"name":                     name,
		"query":                    query,
		"parallelism":              parallelism,
		"checkpointIntervalMicros": 60000000,
		"udfs":                     []any{},
	}

	resp, err := ac.Post("/api/v1/pipelines", payload)
	if err != nil {
		return nil, fmt.Errorf("failed to reach Arroyo API: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("Arroyo returned %d: %s", resp.StatusCode, string(body))
	}

	var out PipelineResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("failed to decode Arroyo response: %w", err)
	}
	return &out, nil
}
