package arroyo

import (
	"encoding/json"
	"net/http"

	"github.com/ntentasd/nostradamus-api/pkg/types"
	"github.com/ntentasd/nostradamus-api/pkg/utils"
)

func (ac *ArroyoClient) ListPipelines(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.ReplyMethodNotAllowed(w)
		return
	}

	defer r.Body.Close()

	resp, err := ac.Get("/api/v1/pipelines")
	if err != nil {
		utils.ReplyJSON(w, http.StatusBadGateway, utils.Body{
			"error": "failed to reach Arroyo API: " + err.Error(),
		})
	}
	defer resp.Body.Close()

	var wrapper struct {
		Data []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"data"`
	}

	if err = json.NewDecoder(resp.Body).Decode(&wrapper); err != nil {
		utils.ReplyJSON(w, http.StatusInternalServerError, utils.Body{
			"error": "failed to read Arroyo response: " + err.Error(),
		})
		return
	}

	utils.ReplyJSON(w, http.StatusOK, utils.Body{
		"data": wrapper.Data,
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
		utils.ReplyJSON(w, http.StatusInternalServerError, utils.Body{
			"error": "failed to read Arroyo response: " + err.Error(),
		})
		return
	}

	utils.ReplyJSON(w, http.StatusOK, utils.Body{
		"data": wrapper.Data,
	})
}
