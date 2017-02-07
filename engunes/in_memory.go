package engunes

import (
	"errors"
	"sync"
	"time"

	"github.com/cafebazaar/hafezieh"
)

type InMemItem struct {
	Item      interface{}
	CreatedAt *time.Time
}

type MemoryCacheConfig struct {
}

type memoryCache struct {
	config MemoryCacheConfig

	items map[string]*InMemItem
	mutex sync.RWMutex
}

func (c *memoryCache) SetRevisitFunc(revisitFunc hafezieh.RevisitFunc) error {
	// TODO
	return errors.New("TODO")
}

func (c *memoryCache) Set(key string, x interface{}, revisitDuration time.Duration) error {
	// TODO
	return nil
}

func (c *memoryCache) Get(key string) (interface{}, error) {
	// TODO
	return nil, hafezieh.MissError
}

func (c *memoryCache) Del(key string) error {
	// TODO
	return nil
}

func NewMemoryCache() hafezieh.Cache {
	return &memoryCache{
		items: make(map[string]*InMemItem),
	}
}
