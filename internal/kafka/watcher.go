package kafka

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/IBM/sarama"
	"github.com/ntentasd/nostradamus-api/internal/arroyo"
)

func NewWatcher(brokers []string, client *arroyo.ArroyoClient, profiles arroyo.ProfileCache) *Watcher {
	return &Watcher{
		brokers:     brokers,
		client:      client,
		knownTopics: make(map[string]bool),
		profiles:    profiles,
	}
}

func (w *Watcher) Run(ctx context.Context) {
	cfg := sarama.NewConfig()
	cfg.Version = sarama.V2_8_0_0
	client, err := sarama.NewClient(w.brokers, cfg)
	if err != nil {
		log.Fatalf("[Watcher] Kafka client: %v", err)
	}
	defer client.Close()

	pipelines, err := w.client.ListPipelinesInternal()
	if err != nil {
		fmt.Printf("[Watcher] failed to list existing pipelines - duplicates expected")
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
				log.Printf("[Watcher] list topics: %v", err)
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

				log.Printf("[Watcher] new topic detected: %s", topic)
				if err := w.createArroyoFlow(topic); err != nil {
					log.Printf("[Watcher] failed to create flow for %s: %v", topic, err)
					continue
				}
				w.knownTopics[topic] = true
			}
			time.Sleep(30 * time.Second)
			if err := client.RefreshMetadata(); err != nil {
				log.Printf("[Watcher] metadata refresh failed: %v", err)
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
			log.Printf("[Watcher] Kafka table %s already exists, skipping", kafkaName)
		} else {
			return fmt.Errorf("[Watcher] failed to create Kafka table %s: %w", kafkaName, err)
		}
	} else {
		log.Printf("[Watcher] Kafka connection table created: %s", kafkaName)
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
			log.Printf("[Watcher] pipeline %s already exists, skipping", topic)
		} else {
			return fmt.Errorf("pipeline creation failed for %s: %v", topic, err)
		}
	}

	log.Printf("[Watcher] created pipeline for %s", topic)
	return nil
}
