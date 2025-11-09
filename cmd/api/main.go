package main

import (
	"context"
	"io"
	stdlog "log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gocql/gocql"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

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
	stdlog.SetOutput(io.Discard)
	log.Logger = zerolog.New(os.Stdout).With().Timestamp().Logger()

	scyllaEnv := os.Getenv("SCYLLA_NODES")
	arroyoURL := os.Getenv("ARROYO_URL")
	kafkaEnv := os.Getenv("KAFKA_BROKERS")

	if scyllaEnv != "" {
		scyllaNodes = strings.Split(scyllaEnv, ",")
	}

	if kafkaEnv != "" {
		kafkaBrokers = strings.Split(kafkaEnv, ",")
	}

	log.Info().Strs("scylla_nodes", scyllaNodes).Msg("parsed Scylla nodes")

	clusterMeta := gocql.NewCluster(scyllaNodes...)
	clusterMeta.Keyspace = "sensors_meta"
	metaSess, err := clusterMeta.CreateSession()
	if err != nil {
		log.Fatal().Err(err).Msg("unable to connect to meta keyspace")
	}

	clusterData := gocql.NewCluster(scyllaNodes...)
	clusterData.Keyspace = "sensors_data"
	dataSess, err := clusterData.CreateSession()
	if err != nil {
		log.Fatal().Err(err).Msg("unable to connect to data keyspace")
	}

	dbLogger := log.Logger.With().Str("component", "db").Logger()
	store := db.New(metaSess, dataSess, dbLogger)
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
		log.Fatal().Msg("VALKEY_NODES or MEMCACHED_NODE must be set")
	}

	if len(valkeyAddrs) > 0 && memcachedAddr != "" {
		log.Fatal().Msg("only one of VALKEY_NODES or MEMCACHED_NODE may be set")
	}

	var c cache.Cache
	if len(valkeyAddrs) > 0 {
		c = cache.NewValkey(valkeyAddrs)
		log.Info().Msg("using Valkey cache driver")
	} else {
		c = cache.NewMemcached(memcachedAddr)
		log.Info().Msg("using Memcached cache driver")
	}
	defer c.Close()

	arroyoLogger := log.Logger.With().Str("component", "arroyo_client").Logger()
	ac := arroyo.New(arroyoURL, arroyoLogger)

	profiles, err := ac.ListConnectionProfiles()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to list Arroyo connection profiles")
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
		log.Fatal().Msg("missing Kafka or Scylla profile")
	}

	log.Info().Str("kafka_profile", pCache.KafkaProfileID).Str("scylla_profile", pCache.ScyllaProfileID).Msg("loaded Arroyo profiles")

	emqxClient := emqx.New()

	appLogger := log.Logger.With().Str("component", "app").Logger()
	app := routes.New(store, c, ac, emqxClient, appLogger)

	shutdown := tracing.InitTracer()
	defer shutdown(context.Background())

	mux := routes.NewMux(app)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	watcherLogger := log.Logger.With().Str("component", "kafka_watcher").Logger()
	watcher := kafka.NewWatcher(kafkaBrokers, ac, pCache, watcherLogger)
	go watcher.Run(ctx)

	supervisorLogger := log.Logger.With().Str("component", "supervisor").Logger()
	sv := worker.NewSupervisor(ac, time.Second*5, supervisorLogger)
	sv.Start(context.Background())
	defer sv.Stop()

	log.Info().Msg("Listening on port :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal().Err(err).Msg("server shutdown")
	}
}
