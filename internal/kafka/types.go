package kafka

import (
	"github.com/ntentasd/nostradamus-api/internal/arroyo"
)

type Watcher struct {
	brokers     []string
	client      *arroyo.ArroyoClient
	knownTopics map[string]bool
	profiles    arroyo.ProfileCache
}
