package hafezieh

import (
	"errors"
	"time"
)

// UseDefaultValue can be used in Set, to use the default revisitDuration value
// by the engine
const UseDefaultValue time.Duration = 0

var (
	// ErrMiss is the error returned on Get, when the key is not available
	ErrMiss = errors.New("Not cached, or deleted")
	// ErrNegativeDuration is the error returned on Set, when revisitDuration<0
	ErrNegativeDuration = errors.New("revisitDuration can't be negative")
)

// Cache is a simple cache interface, to rulw all the cache engunes
type Cache interface {
	// Set stores x and assign it to the key, and if revisitDuration is >0
	// revisit (which is dependant on the engine) happens after revisitDuration
	// from now, unless the key is reset or deleted in the mean time.
	// Some engines may have ability to set default for revisitDuration
	Set(key string, x interface{}, revisitDuration time.Duration) error

	// Returns the assigned object to the key, if is not expired or deleted
	// by now
	Get(key string) (interface{}, error)

	// Deletes the assigned objected
	Del(key string) error

	// Close frees the resources
	Close() error
}
