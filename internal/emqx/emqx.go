package emqx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type EmqxClient struct {
	BaseURL    string
	APIKey     string
	APISecret  string
	HTTPClient *http.Client
}

func New() *EmqxClient {
	url := os.Getenv("EMQX_URL")
	key := os.Getenv("EMQX_API_KEY")
	secret := os.Getenv("EMQX_API_SECRET")

	if url == "" || key == "" || secret == "" {
		panic("missing required EMQX environment variables (EMQX_URL, EMQX_API_KEY, EMQX_API_SECRET)")
	}

	return &EmqxClient{
		BaseURL:   url,
		APIKey:    key,
		APISecret: secret,
		HTTPClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

type CreateUserResponse struct {
	UserID      string `json:"user_id"`
	IsSuperuser bool   `json:"is_superuser"`
}

func (c *EmqxClient) CreateUser(userID, password string, isSuperuser bool) (*CreateUserResponse, error) {
	endpoint := fmt.Sprintf("http://%s/api/v5/authentication/password_based%%3Abuilt_in_database/users", c.BaseURL)

	payload := map[string]any{
		"user_id":      userID,
		"password":     password,
		"is_superuser": isSuperuser,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to encode EMQX payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create EMQX request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(c.APIKey, c.APISecret)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to contact EMQX: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("emqx returned %s: %s", resp.Status, string(b))
	}

	var result CreateUserResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("invalid emqx response: %w", err)
	}

	return &result, nil
}
