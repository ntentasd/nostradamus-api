package routes

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/ntentasd/nostradamus-api/pkg/types"
	"github.com/ntentasd/nostradamus-api/pkg/utils"
)

func (app *App) createPipeline(w http.ResponseWriter, r *http.Request) {
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

	query := `
INSERT INTO temperatures_docker
SELECT
    *
FROM
    kafka2
`

	req := types.Pipeline{
		Name:        body.Name,
		Query:       query,
		Parallelism: 3,
	}

	resp, err := app.client.Post("/api/v1/pipelines", req)
	if err != nil {
		utils.ReplyJSON(w, http.StatusBadGateway, utils.Body{
			"error": "failed to reach Arroyo API: " + err.Error(),
		})
		return
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		utils.ReplyJSON(w, http.StatusInternalServerError, utils.Body{
			"error": "failed to read Arroyo response: " + err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	w.Write(respBytes)
}
