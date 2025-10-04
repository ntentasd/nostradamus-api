package routes

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type ArroyoClient struct {
	BaseURL string
	Client  *http.Client
}

func New(baseURL string) *ArroyoClient {
	return &ArroyoClient{
		BaseURL: baseURL,
		Client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (ac *ArroyoClient) Get(path string) (*http.Response, error) {
	url := fmt.Sprintf("http://%s%s", ac.BaseURL, path)
	return ac.Client.Get(url)
}

func (ac *ArroyoClient) Post(path string, body any) (*http.Response, error) {
	url := fmt.Sprintf("http://%s%s", ac.BaseURL, path)

	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewBuffer(b)
	}

	req, err := http.NewRequest(http.MethodPost, url, reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	return ac.Client.Do(req)
}
