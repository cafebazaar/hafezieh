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

	RevisitNumberOfWorkers int           `mapstructure:"revisit-number-of-workers"`
	RevisitClock           time.Duration `mapstructure:"revisit-clock"`
	RevisitFunc            RevisitFunc

	CleanupMechanism  CleanupMechanism `mapstructure:"cleanup-mechanism"`
	CleanupClock      time.Duration    `mapstructure:"cleanup-clock"`
	CleanupHeapTarget uint64           `mapstructure:"cleanup-heap-target"`
	CleanupPercent    float64          `mapstructure:"cleanup-percent"`
	CleanupCustomFunc CleanupFunc
}

type InMemoryCache struct {
	config MemoryCacheConfig

	items           map[string]*InMemItem
	revisitTimeQMan *revisitTimeQueueManager
	mutex           sync.RWMutex
	stopCleanup     bool
}

func (c *InMemoryCache) Set(key string, x interface{}, revisitDuration time.Duration) error {
	if revisitDuration < 0 {
		return hafezieh.ErrNegativeDuration
	}
	if revisitDuration == 0 {
		revisitDuration = c.config.DefaultRevisitDuration
	}
	if revisitDuration > 0 && revisitDuration < (5*time.Second) {
		return SmallDurationError
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
	c.stopCleanup = true
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

func (c *InMemoryCache) cleanupLoop() {
	var cFunc CleanupFunc
	switch c.config.CleanupMechanism {
	case CleanupNone:
		return
	case CleanupHeapBasedLRU:
		cFunc = heapBasedCleanup
	case CleanupCustomFunc:
		cFunc = c.config.CleanupCustomFunc
	}

	for {
		if c.stopCleanup {
			return
		}
		time.Sleep(c.config.CleanupClock)
		cFunc(c)
	}
}

func validateAndSetDefaults(config *MemoryCacheConfig) error {
	if config.RevisitNumberOfWorkers > 0 {
		if config.RevisitClock == 0 {
			config.RevisitClock = 30 * time.Second
		}
		if config.RevisitFunc == nil {
			return errors.New("No RevisitFunc is set but RevisitNumberOfWorkers is greater than 0")
		}
	}

	if config.CleanupMechanism == CleanupCustomFunc && config.CleanupCustomFunc == nil {
		return errors.New("No CleanupCustomFunc is set but CleanupMechanism is set on CleanupCustomFunc")
	}
	if config.CleanupMechanism != CleanupNone {
		if config.CleanupClock == 0 {
			config.CleanupClock = time.Minute
		}
		if config.CleanupClock < 5*time.Second {
			return errors.New("CleanupClock should be at keast 5 seconds")
		}
		if config.CleanupPercent == 0 {
			config.CleanupPercent = 5
		}
		if config.CleanupPercent > 100 || config.CleanupPercent < 0 {
			return errors.New("CleanupPercent should be between 0 and 100")
		}
	}
	if config.CleanupMechanism == CleanupHeapBasedLRU {
		if config.CleanupHeapTarget == 0 {
			return errors.New("No CleanupHeapTarget is set")
		}
	}

	return nil
}

func NewMemoryCache(config *MemoryCacheConfig) (hafezieh.Cache, error) {
	err := validateAndSetDefaults(config)
	if err != nil {
		return nil, err
	}

	c := &InMemoryCache{
		config: *config,

		items: make(map[string]*InMemItem),
	}
	if c.config.RevisitNumberOfWorkers > 0 {
		c.revisitTimeQMan = initRevisitTimeQueueManager(config.RevisitClock, config.RevisitNumberOfWorkers, c.callRevisit)
	}
	go c.cleanupLoop()

	return c, nil
}
