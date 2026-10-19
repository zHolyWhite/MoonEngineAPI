package cache

import (
	"golang.org/x/sync/singleflight"
)

type Flight struct {
	group singleflight.Group
}

func NewFlight() *Flight {
	return &Flight{}
}

func (f *Flight) Do(key string, fn func() ([]byte, error)) ([]byte, error, bool) {
	value, err, shared := f.group.Do(key, func() (any, error) {
		return fn()
	})
	if err != nil {
		return nil, err, shared
	}
	data, _ := value.([]byte)
	return data, nil, shared
}
