package scheduler

import (
	"fmt"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

type Scheduler struct {
	Tasks        TaskMap
	FuncRegister TaskFunctionRegister

	stopChan chan struct{}
	done     chan struct{}
	interval time.Duration

	once     sync.Once
	mu       sync.Mutex
	started  bool
	stopped  bool
	running  map[TaskID]bool
	taskRuns sync.WaitGroup
}

func NewScheduler() Scheduler {
	return newScheduler(time.Second)
}

func newScheduler(interval time.Duration) Scheduler {
	return Scheduler{
		Tasks:        NewTaskMap(),
		FuncRegister: NewTaskFunctionRegister(),
		stopChan:     make(chan struct{}),
		done:         make(chan struct{}),
		interval:     interval,
		running:      make(map[TaskID]bool),
	}
}

func (scheduler *Scheduler) Start() {
	scheduler.mu.Lock()
	if scheduler.started || scheduler.stopped {
		scheduler.mu.Unlock()
		return
	}
	scheduler.started = true
	scheduler.mu.Unlock()

	go func() {
		ticker := time.NewTicker(scheduler.interval)
		defer ticker.Stop()
		defer close(scheduler.done)
		for {
			select {
			case <-ticker.C:
				scheduler.runPending()
			case <-scheduler.stopChan:
				return
			}
		}
	}()
}

func (Scheduler *Scheduler) Stop() {
	Scheduler.once.Do(func() {
		Scheduler.mu.Lock()
		Scheduler.stopped = true
		started := Scheduler.started
		close(Scheduler.stopChan)
		if !started {
			close(Scheduler.done)
		}
		Scheduler.mu.Unlock()
	})
	<-Scheduler.done
	Scheduler.taskRuns.Wait()
}

func (Scheduler *Scheduler) Wait() {
	<-Scheduler.done
}

func (scheduler *Scheduler) ScheduleAt(time time.Time, function Function, params ...FuncParam) error {
	scheduler.mu.Lock()
	defer scheduler.mu.Unlock()
	if scheduler.stopped {
		return fmt.Errorf("scheduler is stopped")
	}
	funcID, err := scheduler.FuncRegister.AddFunction(function, params...)
	if err != nil {
		return err
	}

	schedule := Schedule{
		IsRecurring: false,
		NextRun:     time,
	}
	scheduler.Tasks.AddTask(schedule, funcID)

	return nil
}

func (scheduler *Scheduler) ScheduleAfter(duration time.Duration, function Function, params ...FuncParam) error {
	return scheduler.ScheduleAt(time.Now().Add(duration), function, params...)
}

func (scheduler *Scheduler) ScheduleEvery(duration time.Duration, function Function, params ...FuncParam) error {
	if duration <= 0 {
		return fmt.Errorf("schedule duration must be positive")
	}
	scheduler.mu.Lock()
	defer scheduler.mu.Unlock()
	if scheduler.stopped {
		return fmt.Errorf("scheduler is stopped")
	}
	funcID, err := scheduler.FuncRegister.AddFunction(function, params...)
	if err != nil {
		return err
	}

	schedule := Schedule{
		IsRecurring: true,
		NextRun:     time.Now().Add(duration),
		Duration:    duration,
	}
	scheduler.Tasks.AddTask(schedule, funcID)

	return nil
}

func (scheduler *Scheduler) runPending() {
	scheduler.mu.Lock()
	now := time.Now()
	for id, task := range scheduler.Tasks {
		if task.IsDue() && !scheduler.running[id] {
			if function, ok := scheduler.FuncRegister.Functions[task.FunctionID]; ok {
				scheduler.running[id] = true
				scheduler.taskRuns.Add(1)
				go scheduler.runTask(id, function)
			}

			if !task.Schedule.IsRecurring {
				delete(scheduler.Tasks, id)
			} else {
				task.Schedule.LastRun = now
				task.Schedule.NextRun = now.Add(task.Schedule.Duration)
			}
		}
	}
	scheduler.mu.Unlock()
}

func (scheduler *Scheduler) runTask(id TaskID, function TaskFunction) {
	defer scheduler.taskRuns.Done()
	defer func() {
		if recovered := recover(); recovered != nil {
			log.Errorf("scheduled task panicked: %v", recovered)
		}
		scheduler.mu.Lock()
		delete(scheduler.running, id)
		scheduler.mu.Unlock()
	}()
	function.Run()
}
