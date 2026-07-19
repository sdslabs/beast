package Taskerpool

import (
	"fmt"
	"runtime"
	"sync"

	log "github.com/sirupsen/logrus"
)

type Queue struct {
	TaskQueue chan Task
	Mux       sync.RWMutex
	InQueue   map[string]bool // A map which stores if the task related to some id is already in the queue

	CompletionChannel chan bool
	stopChannel       chan struct{}
	stopOnce          sync.Once
	workers           sync.WaitGroup
	stopped           bool
	errors            []error
}

func (q *Queue) RecordError(err error) {
	if err == nil {
		return
	}
	q.Mux.Lock()
	defer q.Mux.Unlock()
	q.errors = append(q.errors, err)
}

func (q *Queue) Errors() []error {
	q.Mux.RLock()
	defer q.Mux.RUnlock()
	return append([]error(nil), q.errors...)
}

type Task struct {
	ID   string
	Info interface{}
}

type Worker interface {
	PerformTask(Task) *Task
}

func (q *Queue) Push(w Task) error {
	q.Mux.Lock()
	defer q.Mux.Unlock()
	if q.stopped {
		return fmt.Errorf("queue is stopped")
	}
	if _, ex := q.InQueue[w.ID]; ex {
		log.Warnf("The Task ID : %s is already in queue", w.ID)
		return fmt.Errorf("The Task ID : %s is already in queue", w.ID)
	}
	q.InQueue[w.ID] = true
	select {
	case q.TaskQueue <- w:
	default:
		delete(q.InQueue, w.ID)
		return fmt.Errorf("Queue is full")
	}
	// TODO : get size of the queue
	return nil
}

func (q *Queue) Pop(ID string) {
	q.Mux.Lock()
	delete(q.InQueue, ID)
	completed := q.CompletionChannel != nil && len(q.InQueue) == 0
	q.Mux.Unlock()
	if completed {
		select {
		case q.CompletionChannel <- true:
		default:
		}
	}
}

func (q *Queue) Stop() {
	q.stopOnce.Do(func() {
		q.Mux.Lock()
		q.stopped = true
		close(q.stopChannel)
		q.Mux.Unlock()
		q.workers.Wait()

		q.Mux.Lock()
		for id := range q.InQueue {
			delete(q.InQueue, id)
		}
		q.Mux.Unlock()
	})
}

func (q *Queue) startConcurrentWorker(i int, worker Worker) {
	defer q.workers.Done()
	for {
		select {
		case <-q.stopChannel:
			return
		case w := <-q.TaskQueue:
			newTask := worker.PerformTask(w)

			q.Pop(w.ID)

			if newTask != nil {
				_ = q.Push(*newTask)
			}
		}
	}
}

func (q *Queue) StartWorkers(worker Worker) {
	numCPUs := runtime.NumCPU()
	log.Info("Total Workers: ", numCPUs)
	q.workers.Add(numCPUs)
	for i := 0; i < numCPUs; i++ {
		go q.startConcurrentWorker(i, worker)
	}
}

func InitQueue(maxQueueSize uint32, completionChannel chan bool) *Queue {
	var Q *Queue
	Q = &Queue{
		TaskQueue:         make(chan Task, maxQueueSize),
		Mux:               sync.RWMutex{},
		InQueue:           map[string]bool{},
		CompletionChannel: completionChannel,
		stopChannel:       make(chan struct{}),
	}
	return Q
}
