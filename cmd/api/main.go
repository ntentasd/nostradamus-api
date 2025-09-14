package main

import (
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gocql/gocql"
	"github.com/ntentasd/nostradamus-api/internal/cache"
	"github.com/ntentasd/nostradamus-api/internal/db"
	routes "github.com/ntentasd/nostradamus-api/internal/routes"
)

var (
	scyllaNodes []string
	valkeyNodes []string
)

func main() {
	scyllaEnv := os.Getenv("SCYLLA_NODES")
	valkeyEnv := os.Getenv("VALKEY_NODES")

	if scyllaEnv != "" {
		scyllaNodes = strings.Split(scyllaEnv, ",")
	}

	if valkeyEnv != "" {
		valkeyNodes = strings.Split(valkeyEnv, ",")
	}

	cluster := gocql.NewCluster(scyllaNodes...)
	cluster.Keyspace = "sensors"
	// Remove
	cluster.DisableInitialHostLookup = true
	cluster.DisableShardAwarePort = true

	sess, err := cluster.CreateSession()
	if err != nil {
		log.Fatalf("unable to connect: %v", err)
	}
	defer sess.Close()

	store := db.New(sess)
	defer store.Close()

	cache := cache.New(valkeyNodes...)
	defer cache.Close()

	mux := routes.NewMux(store, cache)

	log.Println("Listening on port :8080...")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		panic(err)
	}
}
