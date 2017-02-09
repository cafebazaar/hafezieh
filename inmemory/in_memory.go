package engunes

import (
	"errors"
	"sync"
	"time"

	"github.com/cafebazaar/hafezieh"
)

var (
	SmallDurationError = errors.New("less than 5 seconds isn't supported by this engine")
)

type InMemItem struct {
	Item       interface{}
	CreatedAt  time.Time
	LastAccess time.Time
	Hits       uint

	revisitTime uint64
}

type MemoryCacheConfig struct {
	DefaultRevisitDuration time.Duration `mapstructure:"default-revisit-duration"`
}

type RevisitFunc func(hafezieh.Cache, string, *InMemItem)

type memoryCache struct {
	config MemoryCacheConfig

	items       map[string]*InMemItem
	mutex       sync.Mutex
	revisitFunc RevisitFunc
}

func (c *memoryCache) Set(key string, x interface{}, revisitDuration time.Duration) error {
	if revisitDuration < 0 {
		return hafezieh.NegativeDurationError
	}
	if revisitDuration > 0 && revisitDuration < (5*time.Second) {
		return SmallDurationError
	}
	if revisitDuration > 0 {
		c.mutex.Lock()
		defer c.mutex.Unlock()
	}
	n := time.Now()
	var revisitTime uint64
	if revisitDuration > 0 {
		revisitTime = uint64(n.Add(revisitDuration).UnixNano())
	}
	c.items[key] = &InMemItem{
		Item:        x,
		CreatedAt:   n,
		LastAccess:  n,
		Hits:        0,
		revisitTime: revisitTime,
	}
	return nil
}

func (c *memoryCache) Get(key string) (interface{}, error) {
	if inMemItem, found := c.items[key]; found {
		inMemItem.LastAccess = time.Now() // Not guaranteed to always increase
		inMemItem.Hits++                  // Not guaranteed to be accurate
		return inMemItem.Item, nil
	}
	return nil, hafezieh.MissError
}

func (c *memoryCache) Del(key string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if inMemItem, found := c.items[key]; found {
		_ = inMemItem // TODO
		delete(c.items, key)
	}
	return nil
}

func NewMemoryCache(config MemoryCacheConfig) hafezieh.Cache {
	return &memoryCache{
		config: config,

		items: make(map[string]*InMemItem),
	}
}
