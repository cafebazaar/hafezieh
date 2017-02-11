package dummy

import (
	"testing"
	"time"

	"github.com/cafebazaar/hafezieh"
)

func TestDummyEngine(t *testing.T) {
	d := NewDummyCache()
	if d == nil {
		t.Fatal("Unexpected nil")
	}
	err := d.Set("t1", 1, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	val, err := d.Get("t1")
	if val != nil || err != hafezieh.ErrMiss {
		t.Fatalf("Unexpected results. val=%v  err=%v", val, err)
	}
	err = d.Del("t1")
	if err != nil {
		t.Fatal(err)
	}
	err = d.Del("t2")
	if err != nil {
		t.Fatal(err)
	}
}
