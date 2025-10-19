package arroyo

type ConnectionProfile struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Connector   string `json:"connector"`
	Description string `json:"description"`
}

type ConnectionTableRequest struct {
	Name              string                `json:"name"`
	Connector         string                `json:"connector"`
	ConnectionProfile string                `json:"connectionProfileId"`
	Config            ConnectionTableConfig `json:"config"`
	Schema            ConnectionTableSchema `json:"schema"`
}

type ConnectionTableConfig struct {
	Type            map[string]string `json:"type"`
	Topic           string            `json:"topic"`
	AutoOffsetReset string            `json:"autoOffsetReset,omitempty"`
	Format          map[string]any    `json:"format"`
}

type ConnectionTableSchema struct {
	Fields     []any          `json:"fields"`
	BadData    map[string]any `json:"badData"`
	Format     map[string]any `json:"format"`
	Definition map[string]any `json:"definition"`
}

type ConnectionTableResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type PipelineResponse struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Query string `json:"query"`
}

type ProfileCache struct {
	KafkaProfileID  string
	ScyllaProfileID string
}
