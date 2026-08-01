package session

import (
	"log"
	"time"

	"github.com/zHolyWhite/MoonEngineAPI/internal/config"
	"github.com/zHolyWhite/MoonEngineAPI/internal/pipeline"
	"github.com/zHolyWhite/MoonEngineAPI/pkg/audio"
)

func (s *PlayerSession) runWorker(cfg *config.Config, client *pipeline.Client) {
	var buffer [][]byte
	silenceTimer := time.NewTimer(cfg.SilenceTimeout)
	<-silenceTimer.C

	for {
		select {
		case chunk := <-s.AudioCh:
			buffer = append(buffer, chunk)
			if !silenceTimer.Stop() {
				select {
				case <-silenceTimer.C:
				default:
				}
			}
			silenceTimer.Reset(cfg.SilenceTimeout)

		case <-silenceTimer.C:
			if len(buffer) > 0 {
				combined := audio.CombinePCM(buffer)
				buffer = buffer[:0]
				go s.processPipeline(cfg, client, combined)
			}

		case <-s.stopCh:
			return
		}
	}
}

func (s *PlayerSession) processPipeline(cfg *config.Config, client *pipeline.Client, pcmData []byte) {
	if !s.limiter.Allow() {
		log.Printf("[%s] rate limited, skipping utterance", s.PlayerID)
		return
	}

	src, dst := s.Langs()

	wavData := audio.PCMToWAV(pcmData, 48000, 1, 16)

	sttResp, err := client.Transcribe(wavData, src)
	if err != nil {
		log.Printf("[%s] STT error: %v", s.PlayerID, err)
		return
	}

	if !pipeline.IsValidTranscription(sttResp, cfg) {
		log.Printf("[%s] transcription filtered: logprob=%.2f no_speech=%.2f text=%q",
			s.PlayerID, sttResp.AvgLogprob, sttResp.NoSpeechProb, sttResp.Text)
		return
	}

	if src == dst {
		s.sendFn(s.PlayerID, sttResp.Text)
		return
	}

	translateResp, err := client.Translate(sttResp.Text, src, dst)
	if err != nil {
		log.Printf("[%s] translate error: %v", s.PlayerID, err)
		return
	}

	s.sendFn(s.PlayerID, translateResp.Translated)
}
