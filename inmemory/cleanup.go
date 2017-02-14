package inmemory

import (
	"runtime"
	"sort"
	"time"

	"github.com/Sirupsen/logrus"
)

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

type lruPairs []*lastAccessKeyPair

func (a lruPairs) Len() int           { return len(a) }
func (a lruPairs) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a lruPairs) Less(i, j int) bool { return a[i].lastAccess < a[j].lastAccess }

func leastRecentlyUsedPairs(pairs lruPairs, k int) lruPairs {
	if k > len(pairs) {
		return pairs
	}
	sort.Sort(pairs)
	return pairs[0:k]
}

func generatePairs(cache *InMemoryCache) lruPairs {
	cache.mutex.RLock()
	n := len(cache.items)
	pairs := make(lruPairs, n)
	j := 0
	for key, i := range cache.items {
		pairs[j] = &lastAccessKeyPair{i.LastAccess.Unix(), key}
		j++
	}
	cache.mutex.RUnlock()
	return pairs
}

func deletePairs(cache *InMemoryCache, pairs lruPairs) {
	cache.mutex.Lock()
	for _, item := range pairs {
		delete(cache.items, item.key)
	}
	cache.mutex.Unlock()
}

func heapBasedLRUCleanup(cache *InMemoryCache) {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	if mem.HeapAlloc > cache.config.CleanupHeapTarget {
		runtime.GC()
	}
	runtime.ReadMemStats(&mem)
	if mem.HeapAlloc > cache.config.CleanupHeapTarget {
		logrus.Debugf("[InMemoryCache:heapBasedLRUCleanup] HeapAlloc=%d - Cleanup is triggered", mem.HeapAlloc)
		pairs := generatePairs(cache)
		n := len(pairs)
		k := (int(float64(n) * cache.config.CleanupPercent * 0.01))
		if k >= n {
			k = n - 1
		}
		if k > 0 {
			pairs = leastRecentlyUsedPairs(pairs, k)
			seconds := time.Now().Unix() - pairs[k-1].lastAccess
			logrus.Debugf("[InMemoryCache:heapBasedLRUCleanup] Rmoving %d items (most recent was accessed %d seconds ago)", k, seconds)
			deletePairs(cache, pairs)
			logrus.Debugf("[InMemoryCache:heapBasedLRUCleanup] Calling GC")
			runtime.GC()
			runtime.ReadMemStats(&mem)
			logrus.Debugf("[InMemoryCache:heapBasedLRUCleanup] New HeapAlloc=%d", mem.HeapAlloc)
		} else {
			logrus.Debugf("[InMemoryCache:heapBasedLRUCleanup] CleanupPercent(%g%%) of %d means no items will be removed", cache.config.CleanupPercent, n)
		}
	}
}

func numberBasedLRUCleanup(cache *InMemoryCache) {
	cache.mutex.RLock()
	n := len(cache.items)
	cache.mutex.RUnlock()
	if n > int(cache.config.CleanupNumberOfItemsTarget) {
		logrus.Debugf("[InMemoryCache:numberBasedLRUCleanup] len(cache.items)=%d - Cleanup is triggered", n)
		pairs := generatePairs(cache)
		n := len(pairs)
		k := n - int(cache.config.CleanupNumberOfItemsTarget)
		pairs = leastRecentlyUsedPairs(pairs, k)
		if k > 0 {
			seconds := time.Now().Unix() - pairs[k-1].lastAccess
			logrus.Debugf("[InMemoryCache:numberBasedLRUCleanup] Rmoving %d items (most recent was accessed %d seconds ago)", k, seconds)
			deletePairs(cache, pairs)
			logrus.Debugf("[InMemoryCache:numberBasedLRUCleanup] Calling GC")
			runtime.GC()
			var mem runtime.MemStats
			runtime.ReadMemStats(&mem)
			logrus.Debugf("[InMemoryCache:numberBasedLRUCleanup] New HeapAlloc=%d", mem.HeapAlloc)
		}
	}
}
