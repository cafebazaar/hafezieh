package inmemory

import (
	"runtime"
	"sort"

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
	LastAccess int64
	key        string
}

type lruPairs []*lastAccessKeyPair

func (a lruPairs) Len() int           { return len(a) }
func (a lruPairs) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a lruPairs) Less(i, j int) bool { return a[i].LastAccess < a[j].LastAccess }

func lruCleanup(c *InMemoryCache) {
	n := len(c.items)
	if float64(n)*c.config.CleanupPercent < 1 {
		return
	}
	pairs := make(lruPairs, 0, n)
	for key, i := range c.items {
		pairs = append(pairs, &lastAccessKeyPair{i.LastAccess.Unix(), key})
	}
	sort.Sort(pairs)
	n = len(pairs)
	k := (int(float64(n) * c.config.CleanupPercent)) - 1
	if k < 0 {
		k = 0
	}
	if k >= n {
		k = n - 1
	}
	logrus.Debugf("[InMemoryCache:lruCleanup] Rmoving %d items (last pair=%v)", k, pairs[k])
	c.mutex.Lock()
	defer c.mutex.Unlock()
	for i := 0; i <= k; i++ {
		delete(c.items, pairs[i].key)
	}
}

func heapBasedCleanup(cache *InMemoryCache) {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	if mem.HeapAlloc > cache.config.CleanupHeapTarget {
		lruCleanup(cache)
	}
}
