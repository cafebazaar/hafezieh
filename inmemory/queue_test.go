package inmemory

import (
	"container/heap"
	"testing"
	"time"
)

func workerNoop(*InMemKey) {}

func TestRevisitTimeQueueManager(t *testing.T) {
	m := initRevisitTimeQueueManager(0, 0, workerNoop)
	m.Push(&InMemKey{"3", time.Date(2000, 1, 1, 1, 3, 1, 0, time.Local)})
	m.Push(&InMemKey{"4", time.Date(2000, 1, 1, 1, 4, 1, 0, time.Local)})
	m.Push(&InMemKey{"2", time.Date(2000, 1, 1, 1, 2, 1, 0, time.Local)})
	m.Push(&InMemKey{"1", time.Date(2000, 1, 1, 1, 1, 1, 0, time.Local)})

	if m.revisitTimeQ[0].key != "1" {
		t.Fatalf("unexpected peek: %v", m.revisitTimeQ[0])
	}
	i1 := heap.Pop(&m.revisitTimeQ).(*InMemKey)
	if i1.key != "1" {
		t.Fatalf("unexpected pop: %v", i1)
	}
	i2 := heap.Pop(&m.revisitTimeQ).(*InMemKey)
	if i2.key != "2" {
		t.Fatalf("unexpected pop: %v", i2)
	}
	i3 := heap.Pop(&m.revisitTimeQ).(*InMemKey)
	if i3.key != "3" {
		t.Fatalf("unexpected pop: %v", i3)
	}
	i4 := heap.Pop(&m.revisitTimeQ).(*InMemKey)
	if i4.key != "4" {
		t.Fatalf("unexpected pop: %v", i4)
	}
	if len(m.revisitTimeQ) != 0 {
		t.Fatalf("unexpected len: %v", m.revisitTimeQ)
	}
}
