package gateway

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/zHolyWhite/MoonEngineAPI/internal/auth"
	"github.com/zHolyWhite/MoonEngineAPI/internal/config"
	"github.com/zHolyWhite/MoonEngineAPI/internal/scheduler"
	"github.com/zHolyWhite/MoonEngineAPI/internal/session"
	"github.com/zHolyWhite/MoonEngineAPI/internal/worker"
	"github.com/zHolyWhite/MoonEngineAPI/pkg/protocol"
)

const (
	audioMagic   = 0x4D45
	audioVersion = 1
	audioTypePCM = 1
	headerSize   = 16
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type streamState struct {
	utteranceID string
	speaker     string
	spoken      string
	targets     []string
	buffer      bytes.Buffer
	lastSeq     uint32
}

type WSHandler struct {
	cfg      *config.Config
	auth     *auth.Authenticator
	sessions *session.Manager
	sched    *scheduler.Scheduler
	stt      *worker.STTClient
	trans    *worker.TranslateClient
	log      *slog.Logger
}

func NewWSHandler(cfg *config.Config, a *auth.Authenticator, sessions *session.Manager, sched *scheduler.Scheduler, stt *worker.STTClient, trans *worker.TranslateClient, log *slog.Logger) *WSHandler {
	return &WSHandler{
		cfg:      cfg,
		auth:     a,
		sessions: sessions,
		sched:    sched,
		stt:      stt,
		trans:    trans,
		log:      log,
	}
}

func (h *WSHandler) Handle(w http.ResponseWriter, r *http.Request) {
	scope, ok := h.auth.Authenticate(r.Header.Get("Authorization"))
	if !ok {
		w.Header().Set("WWW-Authenticate", "Bearer")
		writeError(w, r, protocol.CodeUnauthorized, "credenciales inválidas", 0)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	conn.SetReadLimit(int64(h.cfg.MaxFrameBytes) + headerSize)
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	})

	c := &wsConn{
		conn:     conn,
		scope:    scope,
		streams:  make(map[uint32]*streamState),
		handler:  h,
	}
	defer c.cleanup()

	if !c.handshake() {
		return
	}

	go c.pingLoop()
	c.readLoop()
}

type wsConn struct {
	conn    *websocket.Conn
	mu      sync.Mutex
	scope   string
	session *session.Session
	streams map[uint32]*streamState
	handler *WSHandler
}

func (c *wsConn) handshake() bool {
	var hello protocol.Hello
	if err := c.readJSON(&hello); err != nil {
		return false
	}
	if hello.Type != "hello" || hello.Contract != protocol.ContractVersion {
		c.closeWith(protocol.CodeInvalidRequest, "hello inválido")
		return false
	}

	sessionID := randomID()
	sess, _ := c.handler.sessions.GetOrCreate(c.scope, sessionID)
	c.session = sess

	c.handler.log.Info("sesión abierta", "sessionId", sessionID, "scope", c.scope, "client", hello.ClientName)
	return c.writeJSON(protocol.HelloOK{
		V:         protocol.ContractVersion,
		Type:      "hello.ok",
		SessionID: sessionID,
		Limits: protocol.Limits{
			MaxBodyBytes:  c.handler.cfg.MaxBodyBytes,
			MaxFrameBytes: c.handler.cfg.MaxFrameBytes,
		},
		Audio: protocol.AudioFormat{
			Format:     "pcm_s16le",
			SampleRate: 48000,
			Channels:   1,
		},
	})
}

func (c *wsConn) readLoop() {
	for {
		if err := c.conn.SetReadDeadline(time.Now().Add(60 * time.Second)); err != nil {
			return
		}

		msgType, data, err := c.conn.ReadMessage()
		if err != nil {
			return
		}

		switch msgType {
		case websocket.TextMessage:
			if !c.handleText(data) {
				return
			}
		case websocket.BinaryMessage:
			if !c.handleAudio(data) {
				return
			}
		}
	}
}

func (c *wsConn) handleText(data []byte) bool {
	var envelope struct {
		V    int    `json:"v"`
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		c.closeWith(protocol.CodeInvalidRequest, "JSON inválido")
		return false
	}

	switch envelope.Type {
	case "ping":
		return c.writeJSON(protocol.Pong{V: protocol.ContractVersion, Type: "pong"})
	case "stream.open":
		return c.handleStreamOpen(data)
	case "stream.end":
		return c.handleStreamEnd(data)
	case "stream.cancel":
		return c.handleStreamCancel(data)
	default:
		c.closeWith(protocol.CodeInvalidRequest, "tipo desconocido")
		return false
	}
}

func (c *wsConn) handleStreamOpen(data []byte) bool {
	var msg protocol.StreamOpen
	if err := json.Unmarshal(data, &msg); err != nil {
		c.closeWith(protocol.CodeInvalidRequest, "stream.open inválido")
		return false
	}
	if msg.StreamID == 0 || len(msg.Targets) == 0 {
		c.closeWith(protocol.CodeInvalidRequest, "streamId y targets requeridos")
		return false
	}
	if msg.Audio.Format != "pcm_s16le" || msg.Audio.SampleRate != 48000 || msg.Audio.Channels != 1 {
		c.closeWith(protocol.CodeInvalidRequest, "formato de audio no soportado")
		return false
	}
	if _, exists := c.streams[msg.StreamID]; exists {
		c.closeWith(protocol.CodeInvalidRequest, "streamId duplicado")
		return false
	}

	c.streams[msg.StreamID] = &streamState{
		utteranceID: randomID(),
		speaker:     msg.Speaker,
		spoken:      msg.Spoken,
		targets:     msg.Targets,
	}
	return true
}

func (c *wsConn) handleStreamEnd(data []byte) bool {
	var msg protocol.StreamEnd
	if err := json.Unmarshal(data, &msg); err != nil {
		c.closeWith(protocol.CodeInvalidRequest, "stream.end inválido")
		return false
	}

	st, ok := c.streams[msg.StreamID]
	if !ok {
		return true
	}
	delete(c.streams, msg.StreamID)

	pcm := make([]byte, st.buffer.Len())
	copy(pcm, st.buffer.Bytes())
	if len(pcm) == 0 {
		return true
	}

	task := &scheduler.Task{
		ID:        st.utteranceID,
		Kind:      scheduler.KindVoice,
		SessionID: c.session.ID,
		Run: func(ctx context.Context) error {
			return c.processVoice(ctx, msg.StreamID, st, pcm)
		},
	}

	if err := c.handler.sched.Enqueue(task); err != nil {
		code := protocol.CodeInternal
		switch {
		case errors.Is(err, scheduler.ErrQueueFull):
			code = protocol.CodeQueueFull
		case errors.Is(err, scheduler.ErrExpired):
			code = protocol.CodeExpired
		case errors.Is(err, scheduler.ErrClosed):
			code = protocol.CodeInternal
		default:
			code = protocol.CodeInvalidRequest
		}
		return c.writeJSON(protocol.RequestFailed{
			V:           protocol.ContractVersion,
			Type:        "request.failed",
			SessionID:   c.session.ID,
			StreamID:    msg.StreamID,
			UtteranceID: st.utteranceID,
			Revision:    1,
			Error: protocol.ErrorBody{
				Code:      code,
				Message:   err.Error(),
				Retryable: protocol.IsRetryable(code),
			},
		})
	}
	return true
}

func (c *wsConn) handleStreamCancel(data []byte) bool {
	var msg protocol.StreamCancel
	if err := json.Unmarshal(data, &msg); err != nil {
		c.closeWith(protocol.CodeInvalidRequest, "stream.cancel inválido")
		return false
	}
	delete(c.streams, msg.StreamID)
	return true
}

func (c *wsConn) handleAudio(data []byte) bool {
	if len(data) < headerSize {
		c.closeWith(protocol.CodeInvalidRequest, "frame de audio demasiado corto")
		return false
	}

	magic := binary.BigEndian.Uint16(data[0:2])
	version := data[2]
	frameType := data[3]
	streamID := binary.BigEndian.Uint32(data[4:8])
	seq := binary.BigEndian.Uint32(data[8:12])
	flags := binary.BigEndian.Uint32(data[12:16])

	if magic != audioMagic || version != audioVersion || frameType != audioTypePCM {
		c.closeWith(protocol.CodeInvalidRequest, "frame de audio inválido")
		return false
	}

	st, ok := c.streams[streamID]
	if !ok {
		c.closeWith(protocol.CodeInvalidRequest, "stream desconocido")
		return false
	}
	if seq != st.lastSeq+1 {
		c.closeWith(protocol.CodeInvalidRequest, "secuencia de audio inválida")
		return false
	}
	if st.buffer.Len()+len(data[headerSize:]) > int(c.handler.cfg.MaxBodyBytes) {
		c.closeWith(protocol.CodePayloadTooLarge, "audio demasiado grande")
		return false
	}

	st.lastSeq = seq
	st.buffer.Write(data[headerSize:])

	if flags&1 != 0 {
		end := protocol.StreamEnd{
			V:           protocol.ContractVersion,
			Type:        "stream.end",
			StreamID:    streamID,
			UtteranceID: st.utteranceID,
		}
		payload, err := json.Marshal(end)
		if err != nil {
			return false
		}
		return c.handleStreamEnd(payload)
	}
	return true
}

func (c *wsConn) processVoice(ctx context.Context, streamID uint32, st *streamState, pcm []byte) error {
	requestID := randomID()

	sttResp, err := c.handler.stt.Transcribe(ctx, worker.STTRequest{
		Player:      st.speaker,
		Format:      "pcm_s16le",
		SampleRate:  48000,
		Channels:    1,
		AudioBase64: worker.EncodePCM(pcm),
		RequestID:   requestID,
		Language:    st.spoken,
	})
	if err != nil {
		c.writeJSON(protocol.RequestFailed{
			V:           protocol.ContractVersion,
			Type:        "request.failed",
			SessionID:   c.session.ID,
			StreamID:    streamID,
			UtteranceID: st.utteranceID,
			Revision:    1,
			Error: protocol.ErrorBody{
				Code:      protocol.CodeWorkerUnavailable,
				Message:   err.Error(),
				Retryable: true,
			},
		})
		return nil
	}

	if sttResp.Status == "no_speech" {
		return nil
	}

	if !c.writeJSON(protocol.TranscriptFinal{
		V:           protocol.ContractVersion,
		Type:        "transcript.final",
		SessionID:   c.session.ID,
		StreamID:    streamID,
		UtteranceID: st.utteranceID,
		Revision:    1,
		Text:        sttResp.Text,
		Language:    sttResp.Language,
		Confidence:  sttResp.Confidence,
	}) {
		return errors.New("no se pudo enviar transcript.final")
	}

	if len(st.targets) == 0 {
		return nil
	}

	transResp, err := c.handler.trans.Translate(ctx, worker.TranslateRequest{
		RequestID: requestID,
		Source:    sttResp.Language,
		Targets:   st.targets,
		Text:      sttResp.Text,
		Kind:      "voice",
	})
	if err != nil {
		c.writeJSON(protocol.RequestFailed{
			V:           protocol.ContractVersion,
			Type:        "request.failed",
			SessionID:   c.session.ID,
			StreamID:    streamID,
			UtteranceID: st.utteranceID,
			Revision:    2,
			Error: protocol.ErrorBody{
				Code:      protocol.CodeWorkerUnavailable,
				Message:   err.Error(),
				Retryable: true,
			},
		})
		return nil
	}

	for _, result := range transResp.Results {
		if result.Error != "" {
			continue
		}
		if !c.writeJSON(protocol.TranslationFinal{
			V:           protocol.ContractVersion,
			Type:        "translation.final",
			SessionID:   c.session.ID,
			StreamID:    streamID,
			UtteranceID: st.utteranceID,
			Revision:    2,
			Target:      result.Target,
			Text:        result.Text,
		}) {
			return errors.New("no se pudo enviar translation.final")
		}
	}
	return nil
}

func (c *wsConn) pingLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		c.mu.Lock()
		err := c.conn.WriteMessage(websocket.PingMessage, nil)
		c.mu.Unlock()
		if err != nil {
			return
		}
	}
}

func (c *wsConn) readJSON(v any) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn.ReadJSON(v)
}

func (c *wsConn) writeJSON(v any) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn.WriteJSON(v) == nil
}

func (c *wsConn) closeWith(code protocol.ErrorCode, message string) {
	_ = c.writeJSON(protocol.ErrorResponse{
		Error: protocol.ErrorBody{
			Code:    code,
			Message: message,
		},
	})
	_ = c.conn.Close()
}

func (c *wsConn) cleanup() {
	if c.session != nil {
		c.handler.sessions.Close(c.scope, c.session.ID)
	}
}

func randomID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "00000000000000000000000000000000"
	}
	return hex.EncodeToString(b)
}