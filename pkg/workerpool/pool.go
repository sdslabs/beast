package Taskerpool

import (
	"fmt"
	"runtime"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

type Queue struct {
	TaskQueue chan Task
	Mux       sync.RWMutex
	InQueue   map[string]bool // A map which stores if the task related to some id is already in the queue

	CompletionChannel chan bool
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
	if _, ex := q.InQueue[w.ID]; ex {
		q.Mux.Unlock()
		log.Warnf("The Task ID : %s is already in queue", w.ID)
		return fmt.Errorf("The Task ID : %s is already in queue", w.ID)
	}
	q.InQueue[w.ID] = true
	q.Mux.Unlock()
	select {
	case q.TaskQueue <- w:
	default:
		q.Mux.Lock()
		delete(q.InQueue, w.ID)
		q.Mux.Unlock()
		return fmt.Errorf("Queue is full")
	}
	return nil
}

func (q *Queue) Pop(ID string) {
	q.Mux.Lock()
	delete(q.InQueue, ID)
	shouldSignal := q.CompletionChannel != nil && len(q.InQueue) == 0
	q.Mux.Unlock()
	if shouldSignal {
		q.CompletionChannel <- true
	}
}

func (q *Queue) finishTask(id string, nextTask *Task) {
	q.Mux.Lock()
	delete(q.InQueue, id)
	if nextTask != nil {
		if _, ex := q.InQueue[nextTask.ID]; !ex {
			q.InQueue[nextTask.ID] = true
			select {
			case q.TaskQueue <- *nextTask:
			default:
				delete(q.InQueue, nextTask.ID)
				log.Warnf("Queue is full while re-enqueuing follow-up task %s", nextTask.ID)
			}
		}
	}
	shouldSignal := q.CompletionChannel != nil && len(q.InQueue) == 0
	q.Mux.Unlock()
	if shouldSignal {
		q.CompletionChannel <- true
	}
}

func (q *Queue) Drain(timeout time.Duration) bool {
	if q == nil {
		return true
	}
	deadline := time.Now().Add(timeout)
	for {
		q.Mux.RLock()
		pending := len(q.InQueue)
		q.Mux.RUnlock()
		if pending == 0 {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func (q *Queue) Stop() {
	if q == nil {
		return
	}
	q.Mux.Lock()
	ids := make([]string, 0, len(q.InQueue))
	for id := range q.InQueue {
		ids = append(ids, id)
	}
	q.Mux.Unlock()

	for _, id := range ids {
		q.Pop(id)
	}
}

func (q *Queue) startConcurrentWorker(i int, worker Worker) {
	for {
		w := <-q.TaskQueue
		newTask := worker.PerformTask(w)
		q.finishTask(w.ID, newTask)
	}
}

func (q *Queue) StartWorkers(worker Worker) {
	numCPUs := runtime.NumCPU()
	log.Info("Total Workers: ", numCPUs)
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
	}
	return Q
}
