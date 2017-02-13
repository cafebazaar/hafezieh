package inmemory

import "testing"

func TestLRUItems(t *testing.T) {
	{
		pairs := make(lruPairs, 0)
		newPairs := lruItems(pairs, 20.0)

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

		newPairs := lruItems(pairs, 20.0)

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
