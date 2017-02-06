package engunes

import (
	"errors"
	"time"

	"github.com/cafebazaar/hafezieh"
)

type dummyCache struct{}

func (d *dummyCache) SetRevisitFunc(revisitFunc hafezieh.RevisitFunc) error {
	return errors.New("revisit is not supported by this engine")
}

func (d *dummyCache) Set(key string, x interface{}, revisitDuration time.Duration) error {
	return nil
}

func (d *dummyCache) Get(key string) (interface{}, error) {
	return nil, hafezieh.MissError
}

func (d *dummyCache) Del(key string) error {
	return nil
}

func NewDummyCache() hafezieh.Cache {
	return &dummyCache{}
}
