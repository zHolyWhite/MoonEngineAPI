package pipeline

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"
)

type Transcriber struct {
	baseURL    string
	httpClient *http.Client
}

func NewTranscriber(baseURL string) *Transcriber {
	return &Transcriber{
		baseURL: baseURL,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (t *Transcriber) Transcribe(wavData []byte, lang string) (STTResponse, error) {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)

	fw, err := mw.CreateFormFile("audio", "audio.wav")
	if err != nil {
		return STTResponse{}, fmt.Errorf("transcribe: create form file: %w", err)
	}
	if _, err = fw.Write(wavData); err != nil {
		return STTResponse{}, fmt.Errorf("transcribe: write audio: %w", err)
	}
	if err = mw.WriteField("lang", lang); err != nil {
		return STTResponse{}, fmt.Errorf("transcribe: write lang field: %w", err)
	}
	mw.Close()

	resp, err := t.httpClient.Post(t.baseURL+"/transcribe", mw.FormDataContentType(), &buf)
	if err != nil {
		return STTResponse{}, fmt.Errorf("transcribe: post: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return STTResponse{}, fmt.Errorf("transcribe: flask returned %d: %s", resp.StatusCode, body)
	}

	var result STTResponse
	if err = json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return STTResponse{}, fmt.Errorf("transcribe: decode response: %w", err)
	}
	return result, nil
}
