package arroyo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func (ac *ArroyoClient) CreateConnectionTable(req ConnectionTableRequest) (*ConnectionTableResponse, error) {
	url := fmt.Sprintf("http://%s/api/v1/connection_tables", ac.BaseURL)

	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}

	httpReq, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := ac.Client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("unexpected %d: %s", resp.StatusCode, string(body))
	}

	var out ConnectionTableResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	return &out, nil
}
