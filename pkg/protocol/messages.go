package protocol

type AudioFormat struct {
	Format     string `json:"format"`
	SampleRate int    `json:"sampleRate"`
	Channels   int    `json:"channels"`
}

type Hello struct {
	V          int    `json:"v"`
	Type       string `json:"type"`
	ClientName string `json:"clientName"`
	Contract   int    `json:"contract"`
}

type HelloOK struct {
	V         int         `json:"v"`
	Type      string      `json:"type"`
	SessionID string      `json:"sessionId"`
	Limits    Limits      `json:"limits"`
	Audio     AudioFormat `json:"audio"`
}

type StreamOpen struct {
	V        int         `json:"v"`
	Type     string      `json:"type"`
	StreamID uint32      `json:"streamId"`
	Speaker  string      `json:"speaker"`
	Spoken   string      `json:"spoken"`
	Targets  []string    `json:"targets"`
	Audio    AudioFormat `json:"audio"`
}

type StreamEnd struct {
	V           int    `json:"v"`
	Type        string `json:"type"`
	StreamID    uint32 `json:"streamId"`
	UtteranceID string `json:"utteranceId"`
}

type StreamCancel struct {
	V           int    `json:"v"`
	Type        string `json:"type"`
	StreamID    uint32 `json:"streamId"`
	UtteranceID string `json:"utteranceId"`
	Reason      string `json:"reason,omitempty"`
}

type TranscriptFinal struct {
	V           int     `json:"v"`
	Type        string  `json:"type"`
	SessionID   string  `json:"sessionId"`
	StreamID    uint32  `json:"streamId"`
	UtteranceID string  `json:"utteranceId"`
	Revision    int     `json:"revision"`
	Text        string  `json:"text"`
	Language    string  `json:"language"`
	Confidence  float64 `json:"confidence"`
}

type TranslationFinal struct {
	V           int    `json:"v"`
	Type        string `json:"type"`
	SessionID   string `json:"sessionId"`
	StreamID    uint32 `json:"streamId"`
	UtteranceID string `json:"utteranceId"`
	Revision    int    `json:"revision"`
	Target      string `json:"target"`
	Text        string `json:"text"`
}

type RequestFailed struct {
	V           int       `json:"v"`
	Type        string    `json:"type"`
	SessionID   string    `json:"sessionId"`
	StreamID    uint32    `json:"streamId"`
	UtteranceID string    `json:"utteranceId"`
	Revision    int       `json:"revision"`
	Error       ErrorBody `json:"error"`
}

type Ping struct {
	V    int    `json:"v"`
	Type string `json:"type"`
}

type Pong struct {
	V    int    `json:"v"`
	Type string `json:"type"`
}
