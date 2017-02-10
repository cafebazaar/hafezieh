package inmemory

import "github.com/cafebazaar/hafezieh"

type RevisitFunc func(cache hafezieh.Cache, key string, item *InMemItem)

func ExpireRevisitFunc(cache hafezieh.Cache, key string, item *InMemItem) {
	cache.Del(key)
}
