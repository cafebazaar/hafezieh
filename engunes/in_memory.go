package engunes

import (
	"sync"
	"time"

	"github.com/cafebazaar/hafezieh"
)

type InMemItem struct {
	Item       interface{}
	LastAccess time.Time
}

type MemoryCacheConfig struct {
}

type memoryCache struct {
	config MemoryCacheConfig

	items       map[string]*InMemItem
	mutex       sync.RWMutex
	revisitFunc hafezieh.RevisitFunc
}

func (c *memoryCache) SetRevisitFunc(revisitFunc hafezieh.RevisitFunc) error {
	c.revisitFunc = revisitFunc
	return nil
}

func (c *memoryCache) Set(key string, x interface{}, revisitDuration time.Duration) error {
	c.items[key] = &InMemItem{
		Item:       x,
		LastAccess: time.Now(),
	}
	return nil
}

func (c *memoryCache) Get(key string) (interface{}, error) {
	if inMemItem, found := c.items[key]; found {
		inMemItem.LastAccess = time.Now()
		return inMemItem.Item, nil
	}
	return nil, hafezieh.MissError
}

func (c *memoryCache) Del(key string) error {
	delete(c.items, key)
	return nil
}

func NewMemoryCache(config MemoryCacheConfig) hafezieh.Cache {
	return &memoryCache{
		config: config,

		items: make(map[string]*InMemItem),
	}
}
