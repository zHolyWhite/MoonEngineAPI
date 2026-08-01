package pipeline

import (
	"encoding/binary"
	"math"
)

const (
	silenceRMSThreshold = 200.0
	minSpeechSamples    = 1600
)

func IsSilence(pcm []byte) bool {
	if len(pcm) < 2 {
		return true
	}
	var sum float64
	samples := len(pcm) / 2
	for i := 0; i < len(pcm)-1; i += 2 {
		sample := float64(int16(binary.LittleEndian.Uint16(pcm[i : i+2])))
		sum += sample * sample
	}
	rms := math.Sqrt(sum / float64(samples))
	return rms < silenceRMSThreshold
}

func HasEnoughSpeech(chunks [][]byte) bool {
	totalSamples := 0
	for _, c := range chunks {
		if !IsSilence(c) {
			totalSamples += len(c) / 2
		}
	}
	return totalSamples >= minSpeechSamples
}
