package hafezieh

import (
	"errors"
	"time"
)

var (
	MissError = errors.New("Not cached, or deleted")
)

type CahceItemHandle interface {
	Item() interface{}
}

type CahceItem struct {
	Key string
	CahceItemHandle
}

type RevisitFunc func(Cache, CahceItem)

// Cache is a simple cache interface, to rulw all the cache engunes
type Cache interface {
	// SetRevisitFunc sets or resets the revisit func which is expected to be
	// called after some duration from Set, if the object is not already
	// removed by the cache engine
	SetRevisitFunc(revisitFunc RevisitFunc) error
	Set(key string, x interface{}, revisitDuration time.Duration) error
	Get(key string) (interface{}, error)
	Del(key string) error
}
