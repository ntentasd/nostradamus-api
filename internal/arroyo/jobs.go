package arroyo

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ntentasd/nostradamus-api/pkg/types"
	"github.com/ntentasd/nostradamus-api/pkg/utils"
)

func (ac *ArroyoClient) ListJobs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.ReplyMethodNotAllowed(w)
		return
	}

	resp, err := ac.Get("/api/v1/jobs")
	if err != nil {
		ac.logger.Error().Err(err).Msg("failed to reach Arroyo API")
		utils.ReplyJSON(w, http.StatusBadGateway, utils.Body{
			"error": "failed to reach Arroyo API: " + err.Error(),
		})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		ac.logger.Warn().Err(err).Str("status", resp.Status).Int("status_code", resp.StatusCode).Msg("unexpected status from Arroyo API")
		utils.ReplyJSON(w, resp.StatusCode, utils.Body{
			"error": fmt.Sprintf("Arroyo API returned %d", resp.StatusCode),
		})
		return
	}

	var jobWrapper struct {
		Data []struct {
			ID        string          `json:"id"`
			State     types.StateType `json:"state"`
			StartTime int64           `json:"startTime"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&jobWrapper); err != nil {
		ac.logger.Error().Err(err).Msg("failed to decode Arroyo response")
		utils.ReplyJSON(w, http.StatusInternalServerError, utils.Body{
			"error": "failed to decode Arroyo response: " + err.Error(),
		})
		return
	}

	var jobs []types.Job

	for _, job := range jobWrapper.Data {
		jobs = append(jobs, types.Job{
			ID:        job.ID,
			State:     job.State,
			StartedAt: job.StartTime,
		})
	}

	utils.ReplyJSON(w, http.StatusOK, utils.Body{
		"data": jobs,
	})
}

func (ac *ArroyoClient) GetJob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.ReplyMethodNotAllowed(w)
		return
	}

	id := r.PathValue("id")
	if id == "" {
		utils.ReplyJSON(w, http.StatusBadRequest, utils.Body{
			"error": "missing or invalid pipeline id",
		})
		return
	}

	resp, err := ac.Get(fmt.Sprintf("/api/v1/pipelines/%s/jobs", id))
	if err != nil {
		ac.logger.Error().Err(err).Msg("failed to list jobs")
		utils.ReplyJSON(w, http.StatusInternalServerError, utils.Body{
			"error": err.Error(),
		})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		ac.logger.Warn().Err(err).Msg("unexpected status from Arroyo API")
		utils.ReplyJSON(w, resp.StatusCode, utils.Body{
			"error": fmt.Sprintf("Arroyo API returned %d", resp.StatusCode),
		})
		return
	}

	var pipelineWrapper struct {
		Data []struct {
			ID             string          `json:"id"`
			State          types.StateType `json:"state"`
			RunningDesired bool            `json:"runningDesired"`
			StartTime      int64           `json:"startTime"`
		} `json:"data"`
	}

	err = json.NewDecoder(resp.Body).Decode(&pipelineWrapper)
	if err != nil {
		ac.logger.Error().Err(err).Msg("failed to decode Arroyo response")
		utils.ReplyJSON(w, http.StatusInternalServerError, utils.Body{
			"error": "failed to decode Arroyo response: " + err.Error(),
		})
		return
	}

	switch len(pipelineWrapper.Data) {
	case 0:
		utils.ReplyJSON(w, http.StatusOK, utils.Body{})
	case 1:
		utils.ReplyJSON(w, http.StatusOK, utils.Body{
			"data": pipelineWrapper.Data[0],
		})
	default:
		utils.ReplyJSON(w, http.StatusOK, utils.Body{
			"data": pipelineWrapper.Data,
		})
	}
}
