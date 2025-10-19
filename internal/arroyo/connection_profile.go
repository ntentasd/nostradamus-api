package arroyo

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func (ac *ArroyoClient) ListConnectionProfiles() ([]ConnectionProfile, error) {
	url := fmt.Sprintf("http://%s/api/v1/connection_profiles", ac.BaseURL)

	resp, err := ac.Client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to reach Arroyo API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected %d: %s", resp.StatusCode, string(body))
	}

	var out struct {
		Data []ConnectionProfile `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode profiles: %w", err)
	}

	return out.Data, nil
}
