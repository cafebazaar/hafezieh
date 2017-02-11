package dummy

import (
	"time"

	"github.com/cafebazaar/hafezieh"
)

type dummyCache struct{}

func (c *dummyCache) Set(key string, x interface{}, revisitDuration time.Duration) error {
	return nil
}

func (c *dummyCache) Get(key string) (interface{}, error) {
	return nil, hafezieh.ErrMiss
}

func (c *dummyCache) Del(key string) error {
	return nil
}

func (c *dummyCache) Close() error {
	return nil
}

// NewDummyCache returns a Cache instance which always miss
func NewDummyCache() hafezieh.Cache {
	return &dummyCache{}
}
