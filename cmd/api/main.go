package main

import (
	"log"
	"net/http"

	_ "github.com/gocql/gocql"
	"github.com/ntentasd/nostradamus-api/pkg/utils"
)

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		utils.ReplyJson(w, http.StatusOK, map[string]any{
			"state": "healthy",
		})
	})

	log.Println("Listening on port :8080...")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		panic(err)
	}
}
