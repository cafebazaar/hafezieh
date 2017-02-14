package inmemory

import "testing"
import "fmt"
import "time"

func TestGeneratePairs(t *testing.T) {
	c := &InMemoryCache{items: make(map[string]*InMemItem)}
	n := time.Now()
	for i := 0; i < 10; i++ {
		c.items[fmt.Sprintf("%03d", i)] = &InMemItem{i, n, n.Add(time.Duration(i) * time.Second), 0, 0, nil}
	}
	pairs := generatePairs(c)
	if len(pairs) != 10 {
		t.Fatal("unexpected len:", len(pairs))
	}
	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("%03d", i)
		found := false
		for _, p := range pairs {
			if p.key == key {
				found = true
				expectedLastAccess := n.Add(time.Duration(i) * time.Second).Unix()
				if p.lastAccess != expectedLastAccess {
					t.Fatalf("unexpected lastAccess for items[%s]: %v != %v", key, p.lastAccess, expectedLastAccess)
				}
			}
		}
		if !found {
			t.Fatalf("didn't found items[%s]", key)
		}
	}
}

func TestLeastRecentlyUsedPairs(t *testing.T) {
	{
		pairs := make(lruPairs, 0)
		newPairs := leastRecentlyUsedPairs(pairs, 2)
		if len(newPairs) != 0 {
			t.Fatal("unexpected len(newPairs):", len(newPairs))
		}
	}
	{
		pairs := make(lruPairs, 10)
		pairs[0] = &lastAccessKeyPair{1070, "1070"}
		pairs[1] = &lastAccessKeyPair{1050, "1050"}
		pairs[2] = &lastAccessKeyPair{1030, "1030"}
		pairs[3] = &lastAccessKeyPair{1060, "1060"}
		pairs[4] = &lastAccessKeyPair{1040, "1040"}
		pairs[5] = &lastAccessKeyPair{1080, "1080"}
		pairs[6] = &lastAccessKeyPair{1090, "1090"}
		pairs[7] = &lastAccessKeyPair{1020, "1020"}
		pairs[8] = &lastAccessKeyPair{1000, "1000"}
		pairs[9] = &lastAccessKeyPair{1010, "1010"}
		newPairs := leastRecentlyUsedPairs(pairs, 2)
		if len(newPairs) != 2 {
			t.Fatal("unexpected len(newPairs):", len(newPairs))
		}
		if newPairs[0].lastAccess != 1000 {
			t.Fatal("unexpected newPairs[0]:", newPairs[0].key)
		}
		if newPairs[1].lastAccess != 1010 {
			t.Fatal("unexpected newPairs[1]:", newPairs[1].key)
		}
	}
}

func TestDeletePairs(t *testing.T) {
	c := &InMemoryCache{items: make(map[string]*InMemItem)}
	n := time.Now()
	for i := 0; i < 10; i++ {
		c.items[fmt.Sprintf("%03d", i)] = &InMemItem{i, n, n.Add(time.Duration(i) * time.Second), 0, 0, nil}
	}
	pairs := generatePairs(c)
	deletePairs(c, pairs[:9])
	if len(c.items) != 1 {
		t.Fatal("unexpected len:", len(c.items))
	}
	if _, found := c.items[pairs[9].key]; !found {
		t.Fatalf("didn't found items[%s] - items=%v", pairs[9].key, c.items)
	}
}
