package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gocql/gocql"
	"github.com/ntentasd/nostradamus-api/internal/arroyo"
	"github.com/ntentasd/nostradamus-api/internal/cache"
	"github.com/ntentasd/nostradamus-api/internal/db"
	"github.com/ntentasd/nostradamus-api/internal/emqx"
	"github.com/ntentasd/nostradamus-api/internal/kafka"
	routes "github.com/ntentasd/nostradamus-api/internal/routes"
	"github.com/ntentasd/nostradamus-api/internal/worker"
)

var (
	scyllaNodes  []string
	valkeyNodes  []string
	kafkaBrokers []string
)

func main() {
	scyllaEnv := os.Getenv("SCYLLA_NODES")
	valkeyEnv := os.Getenv("VALKEY_NODES")
	arroyoURL := os.Getenv("ARROYO_URL")
	kafkaEnv := os.Getenv("KAFKA_BROKERS")

	if scyllaEnv != "" {
		scyllaNodes = strings.Split(scyllaEnv, ",")
	}

	if valkeyEnv != "" {
		valkeyNodes = strings.Split(valkeyEnv, ",")
	}

	if kafkaEnv != "" {
		kafkaBrokers = strings.Split(kafkaEnv, ",")
	}

	clusterMeta := gocql.NewCluster(scyllaNodes...)
	clusterMeta.Keyspace = "sensors_meta"
	// Remove
	clusterMeta.DisableInitialHostLookup = true
	clusterMeta.DisableShardAwarePort = true
	metaSess, err := clusterMeta.CreateSession()
	if err != nil {
		log.Fatalf("unable to connect: %v", err)
	}

	clusterData := gocql.NewCluster(scyllaNodes...)
	clusterData.Keyspace = "sensors_data"
	// Remove
	clusterData.DisableInitialHostLookup = true
	clusterData.DisableShardAwarePort = true
	dataSess, err := clusterData.CreateSession()
	if err != nil {
		log.Fatalf("unable to connect: %v", err)
	}

	store := db.New(metaSess, dataSess)
	defer store.Close()

	cache := cache.New(valkeyNodes...)
	defer cache.Close()

	ac := arroyo.New(arroyoURL)

	profiles, err := ac.ListConnectionProfiles()
	if err != nil {
		log.Fatalf("failed to list Arroyo connection profiles: %v", err)
	}

	var pCache arroyo.ProfileCache
	for _, p := range profiles {
		switch p.Connector {
		case "kafka":
			pCache.KafkaProfileID = p.ID
		case "scylla":
			if strings.Contains(p.Name, "docker") {
				pCache.ScyllaProfileID = p.ID
			}
		}
	}

	if pCache.KafkaProfileID == "" || pCache.ScyllaProfileID == "" {
		log.Fatalf("missing Kafka or Scylla profile in Arroyo")
	}

	log.Printf("[Arroyo] Kafka profile: %s | Scylla profile: %s",
		pCache.KafkaProfileID, pCache.ScyllaProfileID)

	emqxClient := emqx.New()

	app := routes.App{
		Store:        store,
		Cache:        cache,
		ArroyoClient: ac,
		EmqxClient:   emqxClient,
	}

	mux := routes.NewMux(&app)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	watcher := kafka.NewWatcher(kafkaBrokers, ac, pCache)
	go watcher.Run(ctx)

	sv := worker.NewSupervisor(ac, time.Second*5)
	sv.Start(context.Background())
	defer sv.Stop()

	log.Println("Listening on port :8080...")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		panic(err)
	}
}
