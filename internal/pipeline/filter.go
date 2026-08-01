package pipeline

import (
	"strings"

	"github.com/zHolyWhite/MoonEngineAPI/internal/config"
)

func IsValidTranscription(resp STTResponse, cfg *config.Config) bool {
	if resp.AvgLogprob < cfg.AvgLogprobMin {
		return false
	}
	if resp.NoSpeechProb > cfg.NoSpeechProbMax {
		return false
	}
	if len(strings.TrimSpace(resp.Text)) < cfg.MinTextLen {
		return false
	}
	return true
}
