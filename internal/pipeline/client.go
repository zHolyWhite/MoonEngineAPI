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

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) Transcribe(wavData []byte, lang string) (STTResponse, error) {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)

	fw, err := mw.CreateFormFile("audio", "audio.wav")
	if err != nil {
		return STTResponse{}, err
	}
	if _, err = fw.Write(wavData); err != nil {
		return STTResponse{}, err
	}
	if err = mw.WriteField("lang", lang); err != nil {
		return STTResponse{}, err
	}
	mw.Close()

	resp, err := c.httpClient.Post(c.baseURL+"/transcribe", mw.FormDataContentType(), &buf)
	if err != nil {
		return STTResponse{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return STTResponse{}, fmt.Errorf("STT %d: %s", resp.StatusCode, body)
	}

	var result STTResponse
	if err = json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return STTResponse{}, err
	}
	return result, nil
}

func (c *Client) Translate(text, src, dst string) (TranslateResponse, error) {
	body, err := json.Marshal(TranslateRequest{Text: text, Source: src, Target: dst})
	if err != nil {
		return TranslateResponse{}, err
	}

	resp, err := c.httpClient.Post(c.baseURL+"/translate", "application/json", bytes.NewReader(body))
	if err != nil {
		return TranslateResponse{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		rbody, _ := io.ReadAll(resp.Body)
		return TranslateResponse{}, fmt.Errorf("translate %d: %s", resp.StatusCode, rbody)
	}

	var result TranslateResponse
	if err = json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return TranslateResponse{}, err
	}
	return result, nil
}
