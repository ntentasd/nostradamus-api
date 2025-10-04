package routes

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ntentasd/nostradamus-api/pkg/types"
	"github.com/ntentasd/nostradamus-api/pkg/utils"
)

func (app *App) listJobs(w http.ResponseWriter, r *http.Request) {
	resp, err := app.client.Get("/api/v1/jobs")
	if err != nil {
		utils.ReplyJSON(w, http.StatusInternalServerError, utils.Body{
			"error": err.Error(),
		})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		utils.ReplyJSON(w, resp.StatusCode, utils.Body{
			"error": fmt.Sprintf("Arroyo API returned %d", resp.StatusCode),
		})
		return
	}

	type job struct {
		ID        string          `json:"id"`
		State     types.StateType `json:"state"`
		StartTime int64           `json:"startTime"`
	}

	var wrapper struct {
		Data []job `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&wrapper); err != nil {
		utils.ReplyJSON(w, http.StatusInternalServerError, utils.Body{
			"error": "failed to decode Arroyo response: " + err.Error(),
		})
		return
	}

	var jobs []types.Job

	for _, job := range wrapper.Data {
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
