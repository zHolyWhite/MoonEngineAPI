package pipeline

type STTResponse struct {
	Text         string  `json:"text"`
	Lang         string  `json:"lang"`
	AvgLogprob   float64 `json:"avg_logprob"`
	NoSpeechProb float64 `json:"no_speech_prob"`
}

type TranslateResponse struct {
	Translated string  `json:"translated"`
	Cached     bool    `json:"cached"`
	TimeMs     int     `json:"time_ms"`
}

type TranslateRequest struct {
	Text   string `json:"text"`
	Source string `json:"source"`
	Target string `json:"target"`
}
