package engunes

import (
	"testing"
	"time"

	"github.com/cafebazaar/hafezieh"
)

func TestMemoryEngine(t *testing.T) {
	d := NewMemoryCache(MemoryCacheConfig{})
	if d == nil {
		t.Fatal("Unexpected nil")
	}
	err := d.Set("t1", 1, time.Minute)
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
	if val != nil || err != hafezieh.MissError {
		t.Fatalf("Unexpected results. val=%v  err=%v", val, err)
	}
	err = d.Del("t2")
	if err != nil {
		t.Fatal(err)
	}
}