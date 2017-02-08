package hafezieh

import (
	"time"
)

// Expire is a pre-difined RevisitFunc which simply deletes the item on revisit
func Expire(cache Cache, cacheItem CahceItem, _ time.Duration) {
	cache.Del(cacheItem.Key)
}
