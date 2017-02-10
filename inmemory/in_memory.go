package inmemory

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

	index       int
	revisitTime *time.Time
}

type MemoryCacheConfig struct {
	DefaultRevisitDuration time.Duration `mapstructure:"default-revisit-duration"`
	NumberOfWorkers        int           `mapstructure:"number-of-workers"`
	Clock                  time.Duration `mapstructure:"clock"`

	revisitFunc RevisitFunc
}

type RevisitFunc func(cache hafezieh.Cache, key string, item *InMemItem)

type memoryCache struct {
	config MemoryCacheConfig

	items            map[string]*InMemItem
	revisitTimeQueue revisitTimeQueueManager
	mutex            sync.Mutex
}

func (c *memoryCache) Set(key string, x interface{}, revisitDuration time.Duration) error {
	if revisitDuration < 0 {
		return hafezieh.NegativeDurationError
	}
	if revisitDuration == 0 {
		revisitDuration = c.config.DefaultRevisitDuration
	}
	if revisitDuration > 0 && revisitDuration < (5*time.Second) {
		return SmallDurationError
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()
	n := time.Now()
	var revisitTime *time.Time
	if revisitDuration > 0 {
		r := n.Add(revisitDuration)
		revisitTime = &r
	}
	c.items[key] = &InMemItem{
		Item:        x,
		CreatedAt:   n,
		LastAccess:  n,
		Hits:        0,
		revisitTime: revisitTime,
	}
	if revisitTime != nil {
		c.revisitTimeQueue.Push(&InMemKey{
			key:         key,
			revisitTime: *revisitTime,
		})
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

func (c *memoryCache) callRevisit(inMemKey *InMemKey) {
	revisitFunc := c.config.revisitFunc
	if revisitFunc == nil {
		return
	}
	if inMemItem, found := c.items[inMemKey.key]; found {
		if inMemItem.revisitTime != nil && *inMemItem.revisitTime == inMemKey.revisitTime {
			// Not an old hook
			revisitFunc(c, inMemKey.key, inMemItem)
		}
	}
}

func NewMemoryCache(config MemoryCacheConfig) (hafezieh.Cache, error) {
	// defaults
	if config.NumberOfWorkers == 0 {
		config.NumberOfWorkers = 1
	}
	if config.Clock == 0 {
		config.Clock = 30 * time.Second
	}

	c := &memoryCache{
		config: config,

		items: make(map[string]*InMemItem),
	}
	if c.config.NumberOfWorkers > 0 {
		c.revisitTimeQueue = initRevisitTimeQueueManager(config.Clock, config.NumberOfWorkers, c.callRevisit)
	}

	return c, nil
}
