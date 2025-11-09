package kafka

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/IBM/sarama"
	"github.com/ntentasd/nostradamus-api/internal/arroyo"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type Watcher struct {
	brokers     []string
	client      *arroyo.ArroyoClient
	knownTopics map[string]bool
	profiles    arroyo.ProfileCache
	logger      zerolog.Logger
}

func NewWatcher(brokers []string, client *arroyo.ArroyoClient, profiles arroyo.ProfileCache, logger zerolog.Logger) *Watcher {
	return &Watcher{
		brokers:     brokers,
		client:      client,
		knownTopics: make(map[string]bool),
		profiles:    profiles,
		logger:      logger,
	}
}

func (w *Watcher) Run(ctx context.Context) {
	cfg := sarama.NewConfig()
	cfg.Version = sarama.V2_8_0_0
	client, err := sarama.NewClient(w.brokers, cfg)
	if err != nil {
		w.logger.Fatal().Err(err).Msg("Kafka client error")
	}
	defer client.Close()

	pipelines, err := w.client.ListPipelinesInternal()
	if err != nil {
		w.logger.Error().Err(err).Msg("failed to list existing pipelines - duplicates expected")
	} else {
		for _, pipeline := range pipelines {
			w.knownTopics[pipeline.Name] = true
		}
	}

	for {
		select {
		case <-ctx.Done():
			return
		default:
			topics, err := client.Topics()
			if err != nil {
				w.logger.Error().Err(err).Msg("failed to list topics")
				time.Sleep(10 * time.Second)
				continue
			}
			for _, topic := range topics {
				if strings.HasPrefix(topic, "__") || !isKafkaManagedTopic(topic) {
					continue
				}
				if w.knownTopics[topic] {
					continue
				}

				log.Info().Str("topic", topic).Msg("new topic detected")
				if err := w.createArroyoFlow(topic); err != nil {
					w.logger.Warn().Str("topic", topic).Msg("failed to create flow")
					continue
				}
				w.knownTopics[topic] = true
			}
			time.Sleep(30 * time.Second)
			if err := client.RefreshMetadata(); err != nil {
				w.logger.Warn().Err(err).Msg("failed to refresh metadata")
			}
		}
	}
}

func (w *Watcher) createArroyoFlow(topic string) error {
	kafkaName := "kafka_" + topic
	scyllaName := "scylla_" + topic

	schema := arroyo.JSONSchema

	kafkaReq := arroyo.ConnectionTableRequest{
		Name:              kafkaName,
		Connector:         "kafka",
		ConnectionProfile: w.profiles.KafkaProfileID,
		Config: arroyo.ConnectionTableConfig{
			Type: map[string]string{
				"offset":    "latest",
				"read_mode": "read_uncommitted",
			},
			Topic:           topic,
			AutoOffsetReset: "earliest",
			Format: map[string]any{
				"json": map[string]any{},
			},
		},
		Schema: arroyo.ConnectionTableSchema{
			Fields:  []any{},
			BadData: map[string]any{"drop": map[string]any{}},
			Format:  map[string]any{"json": map[string]any{}},
			Definition: map[string]any{
				"json_schema": schema,
			},
		},
	}

	if _, err := w.client.CreateConnectionTable(kafkaReq); err != nil {
		if strings.Contains(err.Error(), "already exists") {
			w.logger.Info().Str("connection_table", kafkaName).Msg("Kafka connection table already exists, skipping")
		} else {
			w.logger.Error().Err(err).Str("connection_table", kafkaName).Msg("failed to create Kafka table")
			return fmt.Errorf("failed to create Kafka table: %w", err)
		}
	} else {
		log.Info().Str("connection_table", kafkaName).Msg("Kafka connection table created")
	}

	// TODO: create one scylla connection at start
	// scyllaReq := arroyo.ConnectionTableRequest{
	// 	Name:              scyllaName,
	// 	Connector:         "scylla",
	// 	ConnectionProfile: w.profiles.ScyllaProfileID,
	// 	Config: arroyo.ConnectionTableConfig{
	// 		Type: map[string]string{
	// 			"mode": "insert",
	// 		},
	// 		Format: map[string]any{
	// 			"json": map[string]any{
	// 				"timestampFormat": "rfc3339",
	// 			},
	// 		},
	// 		Extra: map[string]any{
	// 			"connectorType": map[string]any{
	// 				"target": map[string]any{
	// 					"keyspace": "sensors_data",
	// 					"table":    topic,
	// 				},
	// 			},
	// 		},
	// 	},
	// 	Schema: arroyo.ConnectionTableSchema{
	// 		Fields:  []any{},
	// 		BadData: map[string]any{"drop": map[string]any{}},
	// 		Format:  map[string]any{"json": map[string]any{}},
	// 		Definition: map[string]any{
	// 			"json_schema": schema,
	// 		},
	// 	},
	// }

	// if _, err := w.client.CreateConnectionTable(scyllaReq); err != nil {
	// 	if strings.Contains(err.Error(), "already exists") {
	// 		log.Printf("[Watcher] Scylla table %s already exists, skipping", scyllaName)
	// 	} else {
	// 		return fmt.Errorf("[Watcher] failed to create Scylla table %s: %w", scyllaName, err)
	// 	}
	// } else {
	// 	log.Printf("[Watcher] Scylla connection table created: %s", scyllaName)
	// }

	switch {
	case strings.HasPrefix(topic, "temperatures_"):
		scyllaName = "scylla_temperatures"
	case strings.HasPrefix(topic, "humidities_"):
		scyllaName = "scylla_humidities"
	case strings.HasPrefix(topic, "ph_levels_"):
		scyllaName = "scylla_ph_levels"
	}

	query := fmt.Sprintf(`
INSERT INTO %s
SELECT * FROM "%s";
`, scyllaName, kafkaName)

	if _, err := w.client.CreatePipelineInternal(topic, query, 1); err != nil {
		if strings.Contains(err.Error(), "already exists") {
			w.logger.Info().Str("topic", topic).Msg("pipeline already exists, skipping")
		} else {
			w.logger.Error().Err(err).Str("topic", topic).Msg("failed to create pipeline")
			return fmt.Errorf("pipeline creation failed for %s: %v", topic, err)
		}
	}

	w.logger.Info().Str("topic", topic).Msg("pipeline created")
	return nil
}
