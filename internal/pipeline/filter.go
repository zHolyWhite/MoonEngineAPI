package pipeline

import "strings"

func IsValidTranscription(resp FlaskSTTResponse) bool {
	if resp.AvgLogprob < -0.8 {
		return false
	}
	if resp.NoSpeechProb > 0.4 {
		return false
	}
	if len(strings.TrimSpace(resp.Text)) < 3 {
		return false
	}
	return true
}
