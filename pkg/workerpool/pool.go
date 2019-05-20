package workerpool

import (
	"fmt"
	"runtime"
	"sync"

	log "github.com/sirupsen/logrus"
)

type Queue struct {
	WorkQueue chan Work
	Mux       sync.RWMutex
	Set       map[string]bool
}

type Work struct {
	ID   string
	Info interface{}
}

func (q *Queue) CheckPush(w Work) error {
	q.Mux.Lock()
	if _, ex := q.Set[w.ID]; ex {
		q.Mux.Unlock()
		log.Warnf("The Work ID : %s is already in queue", w.ID)
		return fmt.Errorf("The Work ID : %s is already in queue", w.ID)
	}
	q.Set[w.ID] = true
	q.Mux.Unlock()
	q.WorkQueue <- w
	// TODO : get size of the queue
	return nil
}

func (q *Queue) Remove(ID string) {
	q.Mux.Lock()
	delete(q.Set, ID)
	q.Mux.Unlock()
}

func (q *Queue) startConcurrentWorker(i int, action func(Work) *Work) {
	var newWork *Work
	for {
		w := <-q.WorkQueue
		newWork = action(w)

		q.Remove(w.ID)

		if newWork != nil {
			q.CheckPush(*newWork)
		}
	}
}

func (q *Queue) StartWorkers(action func(Work) *Work) {
	numCPUs := runtime.NumCPU()
	log.Info("Total workers: ", numCPUs)
	for i := 0; i < numCPUs; i++ {
		go q.startConcurrentWorker(i, action)
	}
}

func InitQueue(maxQueueSize uint32) *Queue {
	var Q *Queue
	Q = &Queue{
		WorkQueue: make(chan Work, maxQueueSize),
		Mux:       sync.RWMutex{},
		Set:       map[string]bool{},
	}
	return Q
}
