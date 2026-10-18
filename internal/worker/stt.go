package worker

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type STTRequest struct {
	Player      string `json:"player"`
	Format      string `json:"format"`
	SampleRate  int    `json:"sampleRate"`
	Channels    int    `json:"channels"`
	AudioBase64 string `json:"audioBase64"`
	RequestID   string `json:"requestId"`
	Language    string `json:"language,omitempty"`
}

type STTTimings struct {
	AudioMs     int `json:"audioMs"`
	QueueMs     int `json:"queueMs"`
	InferenceMs int `json:"inferenceMs"`
	TotalMs     int `json:"totalMs"`
}

type STTResponse struct {
	Text       string     `json:"text"`
	Confidence float64    `json:"confidence"`
	Language   string     `json:"language"`
	Status     string     `json:"status"`
	Kind       string     `json:"kind"`
	Words      []string   `json:"words"`
	Sentences  []string   `json:"sentences"`
	Timings    STTTimings `json:"timings"`
	RequestID  string     `json:"requestId"`
}

type STTClient struct {
	baseURL string
	client  *http.Client
}

func NewSTTClient(baseURL string, timeout time.Duration) *STTClient {
	return &STTClient{
		baseURL: baseURL,
		client:  &http.Client{Timeout: timeout},
	}
}

func (c *STTClient) Transcribe(ctx context.Context, req STTRequest) (STTResponse, error) {
	var result STTResponse
	if c.baseURL == "" {
		return result, fmt.Errorf("STT no configurado")
	}

	body, err := json.Marshal(req)
	if err != nil {
		return result, fmt.Errorf("stt: marshal: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/transcribe", bytes.NewReader(body))
	if err != nil {
		return result, fmt.Errorf("stt: request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return result, fmt.Errorf("stt: do: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return result, fmt.Errorf("stt: status %d: %s", resp.StatusCode, string(data))
	}

	if err := json.NewDecoder(io.LimitReader(resp.Body, 4*1024*1024)).Decode(&result); err != nil {
		return result, fmt.Errorf("stt: decode: %w", err)
	}
	return result, nil
}

func EncodePCM(pcm []byte) string {
	return base64.StdEncoding.EncodeToString(pcm)
}
