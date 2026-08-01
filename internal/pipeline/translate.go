package pipeline

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Translator struct {
	baseURL    string
	httpClient *http.Client
}

func NewTranslator(baseURL string) *Translator {
	return &Translator{
		baseURL: baseURL,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (t *Translator) Translate(text, src, dst string) (TranslateResponse, error) {
	payload, err := json.Marshal(TranslateRequest{Text: text, Source: src, Target: dst})
	if err != nil {
		return TranslateResponse{}, fmt.Errorf("translate: marshal request: %w", err)
	}

	resp, err := t.httpClient.Post(t.baseURL+"/translate", "application/json", bytes.NewReader(payload))
	if err != nil {
		return TranslateResponse{}, fmt.Errorf("translate: post: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return TranslateResponse{}, fmt.Errorf("translate: flask returned %d: %s", resp.StatusCode, body)
	}

	var result TranslateResponse
	if err = json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return TranslateResponse{}, fmt.Errorf("translate: decode response: %w", err)
	}
	return result, nil
}
