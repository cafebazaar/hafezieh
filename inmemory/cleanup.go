package inmemory

import (
	"errors"
	"fmt"
	"runtime"
	"sort"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
)

type janitor struct {
	config *InMemoryCleanupConfig

	cleanupFunc CleanupFunc
	stopFlag    bool
	wg          sync.WaitGroup
}

type InMemoryCleanupConfig struct {
	Mechanism           CleanupMechanism `mapstructure:"mechanism"`
	Clock               time.Duration    `mapstructure:"clock"`
	HeapTarget          uint64           `mapstructure:"heap-target"`
	NumberOfItemsTarget uint64           `mapstructure:"number-target"`
	Percent             float64          `mapstructure:"percent"`
	CustomFunc          CleanupFunc
}

func (config *InMemoryCleanupConfig) validateAndSetDefaults() error {
	if config.Mechanism == CleanupCustomFunc && config.CustomFunc == nil {
		return errors.New("No CustomFunc is set but Mechanism is set on CleanupCustomFunc")
	}
	if config.Mechanism != CleanupNone {
		if config.Clock == 0 {
			config.Clock = time.Minute
		}
		if config.Clock < 5*time.Second {
			return errors.New("Clock should be at keast 5 seconds")
		}
	}
	if config.Mechanism == CleanupHeapBasedLRU {
		if config.HeapTarget == 0 {
			return errors.New("No HeapTarget is set")
		}
		if config.Percent == 0 {
			config.Percent = 5
		}
		if config.Percent > 100 || config.Percent < 0 {
			return errors.New("Percent should be between 0 and 100")
		}
	} else if config.Mechanism == CleanupNumberBasedLRU {
		if config.NumberOfItemsTarget == 0 {
			return errors.New("No NumberOfItemsTarget is set")
		}
	}
	return nil
}

type CleanupMechanism uint8

const (
	// CleanupNone means no cleanup
	CleanupNone CleanupMechanism = iota
	// CleanupCustomFunc calls the set custom function to cleanup the memory
	CleanupCustomFunc = iota
	// CleanupHeapBasedLRU checks the current HeapAlloc, and if it's higher
	// than the CleanupHeapTarget, CleanupPercent% of the least recently used
	// items will be deleted
	CleanupHeapBasedLRU = iota
	// CleanupNumberBasedLRU checks the current number of items, and if it's
	// higher than the CleanupNumberOfItemsTarget, the least recently used
	// items will be deleted so the number of items became (nearly) equal to
	// CleanupNumberOfItemsTarget
	CleanupNumberBasedLRU = iota
)

type CleanupFunc func(*InMemoryCache)

type lastAccessKeyPair struct {
	lastAccess int64
	key        string
}

type lruPairs []lastAccessKeyPair

func (a lruPairs) Len() int           { return len(a) }
func (a lruPairs) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a lruPairs) Less(i, j int) bool { return a[i].lastAccess < a[j].lastAccess }

func (j *janitor) leastRecentlyUsedPairs(pairs lruPairs, k int) lruPairs {
	if k > len(pairs) {
		return pairs
	}
	sort.Sort(pairs)
	return pairs[0:k]
}

func (j *janitor) generatePairs(cache *InMemoryCache) lruPairs {
	cache.mutex.RLock()
	n := len(cache.items)
	pairs := make(lruPairs, n)
	k := 0
	for key, i := range cache.items {
		pairs[k].lastAccess = i.LastAccess.Unix()
		pairs[k].key = key
		k++
	}
	cache.mutex.RUnlock()
	return pairs
}

func (j *janitor) deletePairs(cache *InMemoryCache, pairs lruPairs) {
	cache.mutex.Lock()
	for _, item := range pairs {
		delete(cache.items, item.key)
	}
	cache.mutex.Unlock()
}

func (j *janitor) heapBasedLRUCleanup(cache *InMemoryCache) {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	if mem.HeapAlloc > j.config.HeapTarget {
		runtime.GC()
	}
	runtime.ReadMemStats(&mem)
	if mem.HeapAlloc > j.config.HeapTarget {
		logrus.Debugf("[InMemoryCache:heapBasedLRUCleanup] HeapAlloc=%d - Cleanup is triggered", mem.HeapAlloc)
		pairs := j.generatePairs(cache)
		n := len(pairs)
		k := (int(float64(n) * j.config.Percent * 0.01))
		if k >= n {
			k = n - 1
		}
		if k > 0 {
			pairs = j.leastRecentlyUsedPairs(pairs, k)
			seconds := time.Now().Unix() - pairs[k-1].lastAccess
			logrus.Debugf("[InMemoryCache:heapBasedLRUCleanup] Rmoving %d items (most recent was accessed %d seconds ago)", k, seconds)
			j.deletePairs(cache, pairs)
			logrus.Debugf("[InMemoryCache:heapBasedLRUCleanup] Calling GC")
			runtime.GC()
			runtime.ReadMemStats(&mem)
			logrus.Debugf("[InMemoryCache:heapBasedLRUCleanup] New HeapAlloc=%d", mem.HeapAlloc)
		} else {
			logrus.Debugf("[InMemoryCache:heapBasedLRUCleanup] CleanupPercent(%g%%) of %d means no items will be removed", j.config.Percent, n)
		}
	}
}

func (j *janitor) numberBasedLRUCleanup(cache *InMemoryCache) {
	cache.mutex.RLock()
	n := len(cache.items)
	cache.mutex.RUnlock()
	if n > int(j.config.NumberOfItemsTarget) {
		logrus.Debugf("[InMemoryCache:numberBasedLRUCleanup] len(cache.items)=%d - Cleanup is triggered", n)
		pairs := j.generatePairs(cache)
		n := len(pairs)
		k := n - int(j.config.NumberOfItemsTarget)
		pairs = j.leastRecentlyUsedPairs(pairs, k)
		if k > 0 {
			seconds := time.Now().Unix() - pairs[k-1].lastAccess
			logrus.Debugf("[InMemoryCache:numberBasedLRUCleanup] Rmoving %d items (most recent was accessed %d seconds ago)", k, seconds)
			j.deletePairs(cache, pairs)
			logrus.Debugf("[InMemoryCache:numberBasedLRUCleanup] Calling GC")
			runtime.GC()
			var mem runtime.MemStats
			runtime.ReadMemStats(&mem)
			logrus.Debugf("[InMemoryCache:numberBasedLRUCleanup] New HeapAlloc=%d", mem.HeapAlloc)
		}
	}
}

func (j *janitor) noopCleanup(cache *InMemoryCache) {}

func (j *janitor) loop(cache *InMemoryCache) {
	defer j.wg.Done()
	for {
		time.Sleep(j.config.Clock)
		if j.stopFlag {
			return
		}
		j.cleanupFunc(cache)
		if j.stopFlag {
			return
		}
	}
}

func (j *janitor) stop() {
	j.stopFlag = true
	j.wg.Wait()
}

func newJanitor(config *InMemoryCleanupConfig, cache *InMemoryCache) (*janitor, error) {
	j := &janitor{
		config: config,
	}
	switch j.config.Mechanism {
	case CleanupNone:
		j.cleanupFunc = j.noopCleanup
	case CleanupHeapBasedLRU:
		j.cleanupFunc = j.heapBasedLRUCleanup
	case CleanupNumberBasedLRU:
		j.cleanupFunc = j.numberBasedLRUCleanup
	case CleanupCustomFunc:
		j.cleanupFunc = j.config.CustomFunc
	default:
		return nil, fmt.Errorf("unknown cleanup mechanism: %v", j.config.Mechanism)
	}
	j.wg.Add(1)
	go j.loop(cache)
	return j, nil
}
