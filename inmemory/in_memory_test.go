package inmemory

import (
	"testing"
	"time"

	"github.com/cafebazaar/hafezieh"
)

func TestMemoryEngine(t *testing.T) {
	d, err := NewMemoryCache(&MemoryCacheConfig{})
	if err != nil {
		t.Fatal(err)
	}
	err = d.Set("t1", 1, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	val, err := d.Get("t1")
	if err != nil {
		t.Fatal(err)
	}
	if val != 1 {
		t.Fatalf("Unexpected results. val=%v", val)
	}
	err = d.Del("t1")
	if err != nil {
		t.Fatal(err)
	}
	val, err = d.Get("t1")
	if val != nil || err != hafezieh.ErrMiss {
		t.Fatalf("Unexpected results. val=%v  err=%v", val, err)
	}
	err = d.Del("t2")
	if err != nil {
		t.Fatal(err)
	}
}

func TestValidateAndSetDefaults(t *testing.T) {
	{
		err := validateAndSetDefaults(&MemoryCacheConfig{
			CleanupMechanism: CleanupHeapBasedLRU,
		})
		if err == nil || err.Error() != "No CleanupHeapTarget is set" {
			t.Fatal("expecting another error, got:", err)
		}
	}
	{
		err := validateAndSetDefaults(&MemoryCacheConfig{
			CleanupMechanism:  CleanupHeapBasedLRU,
			CleanupHeapTarget: 100000,
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	{
		err := validateAndSetDefaults(&MemoryCacheConfig{
			CleanupMechanism: CleanupNumberBasedLRU,
		})
		if err == nil || err.Error() != "No CleanupNumberOfItemsTarget is set" {
			t.Fatal("expecting another error, got:", err)
		}
	}
	{
		err := validateAndSetDefaults(&MemoryCacheConfig{
			CleanupMechanism:           CleanupNumberBasedLRU,
			CleanupNumberOfItemsTarget: 100000,
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}
