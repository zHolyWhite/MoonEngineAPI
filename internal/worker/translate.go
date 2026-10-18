package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type TranslateRequest struct {
	RequestID string   `json:"requestId"`
	Source    string   `json:"source"`
	Targets   []string `json:"targets"`
	Text      string   `json:"text"`
	Kind      string   `json:"kind"`
}

type TranslateResult struct {
	Target        string `json:"target"`
	Text          string `json:"text"`
	Cached        bool   `json:"cached"`
	Model         string `json:"model,omitempty"`
	PromptVersion int    `json:"promptVersion,omitempty"`
	Error         string `json:"error,omitempty"`
}

type TranslateResponse struct {
	RequestID string            `json:"requestId"`
	Source    string            `json:"source"`
	Results   []TranslateResult `json:"results"`
}

type TranslateClient struct {
	baseURL string
	client  *http.Client
}

func NewTranslateClient(baseURL string, timeout time.Duration) *TranslateClient {
	return &TranslateClient{
		baseURL: baseURL,
		client:  &http.Client{Timeout: timeout},
	}
}

func (c *TranslateClient) Translate(ctx context.Context, req TranslateRequest) (TranslateResponse, error) {
	var result TranslateResponse
	if c.baseURL == "" {
		return result, fmt.Errorf("traducción no configurada")
	}

	body, err := json.Marshal(req)
	if err != nil {
		return result, fmt.Errorf("translate: marshal: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/translate", bytes.NewReader(body))
	if err != nil {
		return result, fmt.Errorf("translate: request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return result, fmt.Errorf("translate: do: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return result, fmt.Errorf("translate: status %d: %s", resp.StatusCode, string(data))
	}

	if err := json.NewDecoder(io.LimitReader(resp.Body, 4*1024*1024)).Decode(&result); err != nil {
		return result, fmt.Errorf("translate: decode: %w", err)
	}
	return result, nil
}
