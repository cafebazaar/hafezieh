package inmemory

import "testing"
import "fmt"
import "time"

func TestGeneratePairs(t *testing.T) {
	c := &InMemoryCache{items: make(map[string]*InMemItem)}
	j := &janitor{}
	n := time.Now()
	for i := 0; i < 10; i++ {
		c.items[fmt.Sprintf("%03d", i)] = &InMemItem{i, n, n.Add(time.Duration(i) * time.Second), 0, 0, nil}
	}
	pairs := j.generatePairs(c)
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
	j := &janitor{}
	{
		pairs := make(lruPairs, 10)
		pairs[0] = lastAccessKeyPair{1070, "1070"}
		pairs[1] = lastAccessKeyPair{1050, "1050"}
		pairs[2] = lastAccessKeyPair{1030, "1030"}
		pairs[3] = lastAccessKeyPair{1060, "1060"}
		pairs[4] = lastAccessKeyPair{1040, "1040"}
		pairs[5] = lastAccessKeyPair{1080, "1080"}
		pairs[6] = lastAccessKeyPair{1090, "1090"}
		pairs[7] = lastAccessKeyPair{1020, "1020"}
		pairs[8] = lastAccessKeyPair{1000, "1000"}
		pairs[9] = lastAccessKeyPair{1010, "1010"}
		newPairs := j.leastRecentlyUsedPairs(pairs, 2)
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
	{
		pairs := make(lruPairs, 0)
		newPairs := j.leastRecentlyUsedPairs(pairs, 2)
		if len(newPairs) != 0 {
			t.Fatal("unexpected len(newPairs):", len(newPairs))
		}
	}
}

func TestDeletePairs(t *testing.T) {
	j := &janitor{config: &InMemoryCleanupConfig{NumberOfItemsTarget: 100}}
	c := &InMemoryCache{
		items: make(map[string]*InMemItem),
	}
	n := time.Now()
	for i := 0; i < 10; i++ {
		c.items[fmt.Sprintf("%03d", i)] = &InMemItem{i, n, n.Add(time.Duration(i) * time.Second), 0, 0, nil}
	}
	pairs := j.generatePairs(c)
	j.deletePairs(c, pairs[:9])
	if len(c.items) != 1 {
		t.Fatal("unexpected len:", len(c.items))
	}
	if _, found := c.items[pairs[9].key]; !found {
		t.Fatalf("didn't found items[%s] - items=%v", pairs[9].key, c.items)
	}
}

func TestValidateAndSetDefaults(t *testing.T) {
	{
		err := (&InMemoryCleanupConfig{
			Mechanism: CleanupHeapBasedLRU,
		}).validateAndSetDefaults()
		if err == nil || err.Error() != "No HeapTarget is set" {
			t.Fatal("expecting another error, got:", err)
		}
	}
	{
		err := (&InMemoryCleanupConfig{
			Mechanism:  CleanupHeapBasedLRU,
			HeapTarget: 100000,
		}).validateAndSetDefaults()
		if err != nil {
			t.Fatal(err)
		}
	}
	{
		err := (&InMemoryCleanupConfig{
			Mechanism: CleanupNumberBasedLRU,
		}).validateAndSetDefaults()
		if err == nil || err.Error() != "No NumberOfItemsTarget is set" {
			t.Fatal("expecting another error, got:", err)
		}
	}
	{
		err := (&InMemoryCleanupConfig{
			Mechanism:           CleanupNumberBasedLRU,
			NumberOfItemsTarget: 100000,
		}).validateAndSetDefaults()
		if err != nil {
			t.Fatal(err)
		}
	}
}

func BenchmarkNumberBasedLRUCleanup(b *testing.B) {
	b.ReportAllocs()
	j := &janitor{config: &InMemoryCleanupConfig{NumberOfItemsTarget: 100}}
	c := &InMemoryCache{items: make(map[string]*InMemItem)}
	nw := time.Now()
	for i := 0; i < 100; i++ {
		c.items[fmt.Sprintf("%03d", i)] = &InMemItem{i, nw, nw.Add(time.Duration(i) * time.Second), 0, 0, nil}
	}
	for n := 0; n < b.N; n++ {
		for i := 100; i < 200; i++ {
			c.items[fmt.Sprintf("%03d", i)] = &InMemItem{i, nw, nw.Add(time.Duration(i) * time.Second), 0, 0, nil}
			j.numberBasedLRUCleanup(c)
		}
	}
}
