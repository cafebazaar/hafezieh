package hafezieh

import (
	"errors"
	"time"
)

const UseDefaultValue time.Duration = 0

var (
	MissError             = errors.New("Not cached, or deleted")
	NegativeDurationError = errors.New("revisitDuration can't be negative")
)

// Cache is a simple cache interface, to rulw all the cache engunes
type Cache interface {
	Set(key string, x interface{}, revisitDuration time.Duration) error
	Get(key string) (interface{}, error)
	Del(key string) error
}
