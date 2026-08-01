package gateway

import (
	"encoding/binary"
	"log"
	"net/http"

	"github.com/gorilla/websocket"

	"github.com/zHolyWhite/MoonEngineAPI/internal/config"
	"github.com/zHolyWhite/MoonEngineAPI/internal/session"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type WSHandler struct {
	sm  *session.Manager
	cfg *config.Config
}

func NewWSHandler(sm *session.Manager, cfg *config.Config) *WSHandler {
	return &WSHandler{sm: sm, cfg: cfg}
}

func (h *WSHandler) Handle(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[WS] upgrade error: %v", err)
		return
	}
	defer conn.Close()

	var playerID string
	var srcLang, dstLang string

	sendFn := func(pid, text string) {
		payload := append([]byte(pid+":"), []byte(text)...)
		if err := conn.WriteMessage(websocket.TextMessage, payload); err != nil {
			log.Printf("[WS] write error for %s: %v", pid, err)
		}
	}

	for {
		msgType, data, err := conn.ReadMessage()
		if err != nil {
			if playerID != "" {
				h.sm.Remove(playerID)
			}
			return
		}

		switch msgType {
		case websocket.BinaryMessage:
			if len(data) < 20 {
				continue
			}
			uuid := string(data[:16])
			src := string(data[16:18])
			dst := string(data[18:20])
			pcm := data[20:]

			if playerID == "" {
				playerID = uuid
				srcLang = src
				dstLang = dst
			}

			s := h.sm.GetOrCreate(uuid, src, dst, sendFn)
			select {
			case s.AudioCh <- pcm:
			default:
				log.Printf("[WS] audio buffer full for %s, dropping chunk", uuid)
			}

		case websocket.TextMessage:
			if len(data) < 4 {
				continue
			}
			msgKind := binary.BigEndian.Uint16(data[:2])
			switch msgKind {
			case 0x0001:
				if len(data) >= 22 {
					uid := string(data[2:18])
					newSrc := string(data[18:20])
					newDst := string(data[20:22])
					h.sm.UpdateLangs(uid, newSrc, newDst)
				}
			}
		}

		_ = srcLang
		_ = dstLang
	}
}
