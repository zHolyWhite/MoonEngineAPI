package main

import (
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/websocket"
)

const (
	audioMagic   = 0x4D45
	audioVersion = 1
	audioTypePCM = 1
	headerSize   = 16
)

func main() {
	addr := flag.String("addr", "ws://127.0.0.1:8080/v1/stream", "URL WebSocket")
	token := flag.String("token", "", "API key")
	speaker := flag.String("speaker", "smoke", "nombre del hablante")
	spoken := flag.String("spoken", "", "idioma hablado")
	target := flag.String("target", "es", "idioma destino")
	seconds := flag.Float64("seconds", 1, "duración del tono")
	frequency := flag.Float64("frequency", 440, "frecuencia del tono")
	flag.Parse()

	if *token == "" {
		log.Fatal("falta -token")
	}

	u, err := url.Parse(*addr)
	if err != nil {
		log.Fatalf("URL inválida: %v", err)
	}

	header := map[string][]string{
		"Authorization": {"Bearer " + *token},
	}

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), header)
	if err != nil {
		log.Fatalf("no se pudo conectar: %v", err)
	}
	defer conn.Close()

	done := make(chan struct{})
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	go func() {
		defer close(done)
		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				return
			}
			fmt.Printf("RECIBIDO %s\n", string(data))
		}
	}()

	sendJSON(conn, map[string]any{
		"v":          1,
		"type":       "hello",
		"clientName": "smoketest",
		"contract":   1,
	})

	time.Sleep(200 * time.Millisecond)

	sendJSON(conn, map[string]any{
		"v":        1,
		"type":     "stream.open",
		"streamId": 1,
		"speaker":  *speaker,
		"spoken":   *spoken,
		"targets":  []string{*target},
		"audio": map[string]any{
			"format":     "pcm_s16le",
			"sampleRate": 48000,
			"channels":   1,
		},
	})

	pcm := generateTone(*seconds, *frequency)
	frameSize := 4800
	for offset := 0; offset < len(pcm); offset += frameSize {
		end := offset + frameSize
		if end > len(pcm) {
			end = len(pcm)
		}
		last := 0
		if end == len(pcm) {
			last = 1
		}
		sendAudio(conn, 1, uint32(offset/frameSize)+1, last, pcm[offset:end])
		time.Sleep(20 * time.Millisecond)
	}

	sendJSON(conn, map[string]any{
		"v":           1,
		"type":        "stream.end",
		"streamId":    1,
		"utteranceId": "smoke-utterance",
	})

	select {
	case <-done:
	case <-interrupt:
	}

	time.Sleep(500 * time.Millisecond)
}

func generateTone(seconds, frequency float64) []byte {
	samples := int(seconds * 48000)
	pcm := make([]byte, samples*2)
	for i := 0; i < samples; i++ {
		value := int16(math.Sin(2*math.Pi*frequency*float64(i)/48000) * 8000)
		binary.LittleEndian.PutUint16(pcm[i*2:], uint16(value))
	}
	return pcm
}

func sendJSON(conn *websocket.Conn, payload any) {
	data, err := json.Marshal(payload)
	if err != nil {
		log.Fatalf("no se pudo serializar: %v", err)
	}
	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		log.Fatalf("no se pudo enviar: %v", err)
	}
	fmt.Printf("ENVIADO %s\n", string(data))
}

func sendAudio(conn *websocket.Conn, streamID uint32, seq uint32, last int, pcm []byte) {
	frame := make([]byte, headerSize+len(pcm))
	binary.BigEndian.PutUint16(frame[0:2], audioMagic)
	frame[2] = audioVersion
	frame[3] = audioTypePCM
	binary.BigEndian.PutUint32(frame[4:8], streamID)
	binary.BigEndian.PutUint32(frame[8:12], seq)
	binary.BigEndian.PutUint32(frame[12:16], uint32(last))
	copy(frame[headerSize:], pcm)

	if err := conn.WriteMessage(websocket.BinaryMessage, frame); err != nil {
		log.Fatalf("no se pudo enviar audio: %v", err)
	}
}
