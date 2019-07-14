package scheduler

import (
	"time"
)

type Scheduler struct {
	Tasks        TaskMap
	FuncRegister TaskFunctionRegister

	stopChan chan bool
	ticker   *time.Ticker
}

func NewScheduler() Scheduler {
	return Scheduler{
		Tasks:        NewTaskMap(),
		FuncRegister: NewTaskFunctionRegister(),

		stopChan: make(chan bool),
		ticker:   time.NewTicker(1 * time.Second),
	}
}

func (scheduler *Scheduler) Start() {
	go func() {
		for {
			select {
			case <-scheduler.ticker.C:
				scheduler.runPending()
			case <-scheduler.stopChan:
				scheduler.ticker.Stop()
				close(scheduler.stopChan)
			}
		}
	}()
}

func (Scheduler *Scheduler) Stop() {
	Scheduler.stopChan <- true
}

func (Scheduler *Scheduler) Wait() {
	<-Scheduler.stopChan
}

func (scheduler *Scheduler) ScheduleAt(time time.Time, function Function, params ...FuncParam) error {
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
	for id, task := range scheduler.Tasks {
		if task.IsDue() {
			if function, ok := scheduler.FuncRegister.Functions[task.FunctionID]; ok {
				go function.Run()
			}

			if !task.Schedule.IsRecurring {
				delete(scheduler.Tasks, id)
			} else {
				task.Schedule.LastRun = time.Now()
				task.Schedule.NextRun = task.Schedule.NextRun.Add(task.Schedule.Duration)
			}
		}
	}
}
