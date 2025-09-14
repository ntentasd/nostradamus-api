package routes

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/ntentasd/nostradamus-api/pkg/utils"
)

type Body map[string]any

func healthHandler(w http.ResponseWriter, r *http.Request) {
	utils.ReplyJSON(w, http.StatusOK, Body{
		"state": "healthy",
	})
}

func (app *App) latestHandler(w http.ResponseWriter, r *http.Request) {
	year, month, day := time.Now().Date()
	today := fmt.Sprintf("%d-%02d-%02d", year, month, day)
	res, err := app.cache.FetchLast5(
		fmt.Sprintf("%s:%s:%s", "sensor", "temp-sensor-1", today),
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
		log.Println("Falling back to database lookup")
		res, err = app.store.GetLast5Values("temp-sensor-1", today)
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
			app.cache.Store(
				fmt.Sprintf("%s:%s:%s", "sensor", "temp-sensor-1", today),
				entry,
			)
		}
	}

	utils.ReplyJSON(w, http.StatusOK, Body{
		"data": res,
	})
}
