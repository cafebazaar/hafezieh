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
	// CleanupHeapBasedLRU checks the current HeapAlloc, and if it's higher than
	// the CleanupHeapTarget, CleanupPercent% of the items with lowest
	CleanupHeapBasedLRU = iota
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

func lruItems(pairs lruPairs, percent float64) lruPairs {
	sort.Sort(pairs)
	n := len(pairs)
	k := (int(float64(n) * percent * 0.01))
	if k >= n {
		k = n - 1
	}
	return pairs[0:k]
}

func heapBasedCleanup(cache *InMemoryCache) {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	if mem.HeapAlloc > cache.config.CleanupHeapTarget {
		logrus.Debugf("[InMemoryCache:lruCleanup] HeapAlloc=%d - Cleanup is triggered", mem.HeapAlloc)

		cache.mutex.RLock()
		n := len(cache.items)
		pairs := make(lruPairs, n)
		j := 0
		for key, i := range cache.items {
			pairs[j] = &lastAccessKeyPair{i.LastAccess.Unix(), key}
			j++
		}
		cache.mutex.RUnlock()

		lruItems(pairs, cache.config.CleanupPercent)
		k := len(pairs)
		seconds := time.Now().Unix() - pairs[k-1].lastAccess
		logrus.Debugf("[InMemoryCache:lruCleanup] Rmoving %d items (most recent was accessed %d seconds ago)", k, seconds)

		cache.mutex.Lock()
		for _, item := range pairs {
			delete(cache.items, item.key)
		}
		cache.mutex.Unlock()

		logrus.Debugf("[InMemoryCache:lruCleanup] Calling GC")
		runtime.GC()
		runtime.ReadMemStats(&mem)
		logrus.Debugf("[InMemoryCache:lruCleanup] New HeapAlloc=%d", mem.HeapAlloc)
	}
}
