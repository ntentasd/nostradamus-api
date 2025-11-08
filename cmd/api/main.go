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
	"github.com/ntentasd/nostradamus-api/internal/tracing"
	"github.com/ntentasd/nostradamus-api/internal/worker"
)

var (
	scyllaNodes  []string
	kafkaBrokers []string
)

func main() {
	scyllaEnv := os.Getenv("SCYLLA_NODES")
	arroyoURL := os.Getenv("ARROYO_URL")
	kafkaEnv := os.Getenv("KAFKA_BROKERS")

	if scyllaEnv != "" {
		scyllaNodes = strings.Split(scyllaEnv, ",")
	}

	if kafkaEnv != "" {
		kafkaBrokers = strings.Split(kafkaEnv, ",")
	}

	log.Printf("Scylla nodes parsed: %+v", scyllaNodes)

	clusterMeta := gocql.NewCluster(scyllaNodes...)
	clusterMeta.Keyspace = "sensors_meta"
	metaSess, err := clusterMeta.CreateSession()
	if err != nil {
		log.Fatalf("unable to connect: %v", err)
	}

	clusterData := gocql.NewCluster(scyllaNodes...)
	clusterData.Keyspace = "sensors_data"
	dataSess, err := clusterData.CreateSession()
	if err != nil {
		log.Fatalf("unable to connect: %v", err)
	}

	store := db.New(metaSess, dataSess)
	defer store.Close()

	var valkeyAddrs []string
	if nodes := os.Getenv("VALKEY_NODES"); nodes != "" {
		valkeyAddrs = strings.Split(nodes, ",")
	}

	var memcachedAddr string
	if node := os.Getenv("MEMCACHED_NODE"); node != "" {
		memcachedAddr = node
	}

	if len(valkeyAddrs) == 0 && memcachedAddr == "" {
		log.Fatal("either VALKEY_NODES or MEMCACHED_NODE must be set")
	}

	if len(valkeyAddrs) > 0 && memcachedAddr != "" {
		log.Fatal("both VALKEY_NODES and MEMCACHED_NODE are set — only one is allowed")
	}

	var c cache.Cache
	if len(valkeyAddrs) > 0 {
		c = cache.NewValkey(valkeyAddrs)
	} else {
		c = cache.NewMemcached(memcachedAddr)
	}
	defer c.Close()

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
			pCache.ScyllaProfileID = p.ID
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
		Cache:        c,
		ArroyoClient: ac,
		EmqxClient:   emqxClient,
	}

	shutdown := tracing.InitTracer()
	defer shutdown(context.Background())

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
