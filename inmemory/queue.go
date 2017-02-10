package inmemory

import "container/heap"
import "time"

type InMemKey struct {
	key         string
	revisitTime time.Time
}

type revisitTimeQueue []*InMemKey

func (pq revisitTimeQueue) Len() int { return len(pq) }

func (pq revisitTimeQueue) Less(i, j int) bool {
	return pq[i].revisitTime.Before(pq[j].revisitTime)
}

func (pq revisitTimeQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
}

func (pq *revisitTimeQueue) Push(x interface{}) {
	item := x.(*InMemKey)
	*pq = append(*pq, item)
}

func (pq *revisitTimeQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	*pq = old[0 : n-1]
	return item
}

type revisitTimeQueueManager struct {
	revisitTimeQ revisitTimeQueue
	jobs         chan *InMemKey

	clock time.Duration
}

func (m *revisitTimeQueueManager) Push(inMemKey *InMemKey) {
	heap.Push(&m.revisitTimeQ, inMemKey)
}

// Not designed to be run in parallel
func (m *revisitTimeQueueManager) assignLoop() {
	for {
		if len(m.revisitTimeQ) == 0 {
			time.Sleep(m.clock)
			continue
		}
		now := time.Now()
		currentNext := m.revisitTimeQ[0]
		if currentNext.revisitTime.Sub(now) < m.clock {
			m.jobs <- heap.Pop(&m.revisitTimeQ).(*InMemKey)
			continue
		}
		time.Sleep(m.clock)
	}
}

func startWorker(jobs <-chan *InMemKey, worker func(*InMemKey)) {
	for j := range jobs {
		worker(j)
	}
}

func initRevisitTimeQueueManager(clock time.Duration, workerNum int, worker func(*InMemKey)) revisitTimeQueueManager {
	if clock < time.Second {
		clock = time.Second
	}

	manager := revisitTimeQueueManager{
		revisitTimeQ: revisitTimeQueue{},

		clock: clock,
	}
	heap.Init(&manager.revisitTimeQ)
	for i := 0; i < workerNum; i++ {
		go startWorker(manager.jobs, worker)
	}
	go manager.assignLoop()
	return manager
}
