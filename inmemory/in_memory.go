package inmemory

import (
	"errors"
	"sync"
	"time"

	"github.com/cafebazaar/hafezieh"
)

var (
	ErrSmallDuration = errors.New("less than 5 seconds isn't supported by this engine")
)

type InMemoryCache struct {
	config *InMemoryCacheConfig

	items           map[string]*InMemItem
	revisitTimeQMan *revisitTimeQueueManager
	mutex           sync.RWMutex
	janitor         *janitor
}

type InMemoryCacheConfig struct {
	RevisitDefaultDuration time.Duration `mapstructure:"revisit-default-duration"`
	RevisitNumberOfWorkers int           `mapstructure:"revisit-number-of-workers"`
	RevisitClock           time.Duration `mapstructure:"revisit-clock"`
	RevisitFunc            RevisitFunc

	Cleanup *InMemoryCleanupConfig `mapstructure:"cleanup"`
}

func (config *InMemoryCacheConfig) validateAndSetDefaults() error {
	if config.RevisitNumberOfWorkers > 0 {
		if config.RevisitClock == 0 {
			config.RevisitClock = 30 * time.Second
		}
		if config.RevisitFunc == nil {
			return errors.New("No RevisitFunc is set but RevisitNumberOfWorkers is greater than 0")
		}
	}

	if config.Cleanup != nil {
		err := config.Cleanup.validateAndSetDefaults()
		if err != nil {
			return err
		}
	}

	return nil
}

type InMemItem struct {
	Item       interface{}
	CreatedAt  time.Time
	LastAccess time.Time
	Hits       uint

	index       int
	revisitTime *time.Time
}

func (c *InMemoryCache) Set(key string, x interface{}, revisitDuration time.Duration) error {
	if revisitDuration < 0 {
		return hafezieh.ErrNegativeDuration
	}
	if revisitDuration == hafezieh.UseDefaultValue {
		revisitDuration = c.config.RevisitDefaultDuration
	}
	if revisitDuration > 0 && revisitDuration < (5*time.Second) {
		return ErrSmallDuration
	}

	c.mutex.Lock()
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
	if c.revisitTimeQMan != nil && revisitTime != nil {
		c.revisitTimeQMan.Push(&InMemKey{
			key:         key,
			revisitTime: *revisitTime,
		})
	}
	c.mutex.Unlock()
	return nil
}

func (c *InMemoryCache) Get(key string) (interface{}, error) {
	c.mutex.RLock()
	inMemItem, found := c.items[key]
	c.mutex.RUnlock()
	if found {
		inMemItem.LastAccess = time.Now() // Not guaranteed to always increase
		inMemItem.Hits++                  // Not guaranteed to be accurate
		return inMemItem.Item, nil
	}
	return nil, hafezieh.ErrMiss
}

func (c *InMemoryCache) Del(key string) error {
	c.mutex.Lock()
	delete(c.items, key)
	c.mutex.Unlock()
	return nil
}

func (c *InMemoryCache) Close() error {
	if c.revisitTimeQMan != nil {
		c.revisitTimeQMan.Close()
	}
	c.janitor.stop()
	return nil
}

func (c *InMemoryCache) callRevisit(inMemKey *InMemKey) {
	revisitFunc := c.config.RevisitFunc
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

func NewMemoryCache(config *InMemoryCacheConfig) (hafezieh.Cache, error) {
	err := config.validateAndSetDefaults()
	if err != nil {
		return nil, err
	}

	c := &InMemoryCache{
		config: config,

		items: make(map[string]*InMemItem),
	}
	if c.config.RevisitNumberOfWorkers > 0 {
		c.revisitTimeQMan = initRevisitTimeQueueManager(config.RevisitClock, config.RevisitNumberOfWorkers, c.callRevisit)
	}

	if c.config.Cleanup != nil {
		c.janitor, err = newJanitor(c.config.Cleanup, c)
		if err != nil {
			return nil, err
		}
	}

	return c, nil
}
