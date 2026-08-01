package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	ListenAddr      string
	FlaskBaseURL    string
	SilenceTimeout  time.Duration
	RateBurst       int
	RateEvery       time.Duration
	AudioChanCap    int
	AvgLogprobMin   float64
	NoSpeechProbMax float64
	MinTextLen      int
}

func Load() *Config {
	return &Config{
		ListenAddr:      getEnv("LISTEN_ADDR", ":8080"),
		FlaskBaseURL:    getEnv("FLASK_BASE_URL", "http://localhost:5001"),
		SilenceTimeout:  getDurationEnv("SILENCE_TIMEOUT_MS", 600*time.Millisecond),
		RateBurst:       getIntEnv("RATE_BURST", 3),
		RateEvery:       getDurationEnv("RATE_EVERY_MS", 2000*time.Millisecond),
		AudioChanCap:    getIntEnv("AUDIO_CHAN_CAP", 50),
		AvgLogprobMin:   getFloatEnv("AVG_LOGPROB_MIN", -0.8),
		NoSpeechProbMax: getFloatEnv("NO_SPEECH_PROB_MAX", 0.4),
		MinTextLen:      getIntEnv("MIN_TEXT_LEN", 3),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getIntEnv(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}

func getFloatEnv(key string, fallback float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return fallback
}

func getDurationEnv(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if ms, err := strconv.Atoi(v); err == nil {
			return time.Duration(ms) * time.Millisecond
		}
	}
	return fallback
}
