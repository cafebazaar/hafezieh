package inmemory

import (
	"container/heap"
	"sync"
	"time"
)

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
	clock        time.Duration
	jobs         chan *InMemKey
	stop         bool
	wg           sync.WaitGroup
}

func (m *revisitTimeQueueManager) Push(inMemKey *InMemKey) {
	heap.Push(&m.revisitTimeQ, inMemKey)
}

func (m *revisitTimeQueueManager) Close() {
	m.stop = true
	m.wg.Wait()
}

// Not designed to be run in parallel
func (m *revisitTimeQueueManager) assignLoop(mutex *sync.RWMutex) {
	for {
		if m.stop {
			break
		}
		mutex.RLock()
		n := len(m.revisitTimeQ)
		var currentNext *InMemKey
		if n == 0 {
			mutex.RUnlock()
			time.Sleep(m.clock)
			continue
		} else {
			currentNext = m.revisitTimeQ[0]
			mutex.RUnlock()
		}
		now := time.Now()
		if currentNext.revisitTime.Sub(now) < m.clock {
			mutex.Lock()
			m.jobs <- heap.Pop(&m.revisitTimeQ).(*InMemKey)
			mutex.Unlock()
			continue
		}
		if m.stop {
			break
		}
		time.Sleep(m.clock)
	}
	close(m.jobs)
	m.wg.Done()
}

func (m *revisitTimeQueueManager) startWorker(jobs <-chan *InMemKey, worker func(*InMemKey)) {
	for j := range jobs {
		worker(j)
	}
	m.wg.Done()
}

func initRevisitTimeQueueManager(
	mutex *sync.RWMutex, clock time.Duration, workerNum int, worker func(*InMemKey)) *revisitTimeQueueManager {
	if clock < time.Second {
		clock = time.Second
	}

	manager := &revisitTimeQueueManager{
		revisitTimeQ: revisitTimeQueue{},
		clock:        clock,
		jobs:         make(chan *InMemKey, workerNum),
	}
	heap.Init(&manager.revisitTimeQ)
	for i := 0; i < workerNum; i++ {
		manager.wg.Add(1)
		go manager.startWorker(manager.jobs, worker)
	}
	manager.wg.Add(1)
	go manager.assignLoop(mutex)
	return manager
}
