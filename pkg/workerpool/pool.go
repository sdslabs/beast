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
		return fmt.Errorf("Queue is full")
	}
	// TODO : get size of the queue
	return nil
}

func (q *Queue) Pop(ID string) {
	q.Mux.Lock()
	delete(q.InQueue, ID)
	q.Mux.Unlock()
}

func (q *Queue) startConcurrentWorker(i int, worker Worker) {
	var newTask *Task
	for {
		w := <-q.TaskQueue
		newTask = worker.PerformTask(w)

		q.Pop(w.ID)

		if newTask != nil {
			q.Push(*newTask)
		}
	}
}

func (q *Queue) StartWorkers(worker Worker) {
	numCPUs := runtime.NumCPU()
	log.Info("Total Workers: ", numCPUs)
	for i := 0; i < numCPUs; i++ {
		go q.startConcurrentWorker(i, worker)
	}
}

func InitQueue(maxQueueSize uint32) *Queue {
	var Q *Queue
	Q = &Queue{
		TaskQueue: make(chan Task, maxQueueSize),
		Mux:       sync.RWMutex{},
		InQueue:   map[string]bool{},
	}
	return Q
}
