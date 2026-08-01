package ratelimit

import (
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type PlayerLimiter struct {
	limiters sync.Map
	every    time.Duration
	burst    int
}

func NewPlayerLimiter(every time.Duration, burst int) *PlayerLimiter {
	return &PlayerLimiter{
		every: every,
		burst: burst,
	}
}

func (pl *PlayerLimiter) Allow(playerID string) bool {
	v, _ := pl.limiters.LoadOrStore(playerID, rate.NewLimiter(rate.Every(pl.every), pl.burst))
	return v.(*rate.Limiter).Allow()
}

func (pl *PlayerLimiter) Remove(playerID string) {
	pl.limiters.Delete(playerID)
}
