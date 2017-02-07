package hafezieh

// Expire is a pre-difined RevisitFunc which simply deletes the item on revisit
func Expire(cache Cache, cacheItem CahceItem) {
	cache.Del(cacheItem.Key)
}
