package Schedular

import (
	"reflect"
	"fmt"
	"time"
)

type Schedular struct {
	Tasks TaskMap
	FuncRegister TaskFunctionRegister
	
	stopChan     chan bool
	ticker *time.Ticker
}

func NewSchedular() Scheduler {
	register := NewTaskFunctionRegister()
	return Scheduler{
		Tasks: 			NewTaskMap(),
		FuncRegister: NewTaskFunctionRegister(),

		stopChan:     make(chan bool),
		ticker: time.NewTicker(1 * time.Second)
	}
}

func (scheduler *Scheduler) Start() error {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		for {
			select {
			case <-schedular.ticker.C:
				scheduler.runPending()
			case <-sigChan:
				scheduler.stopChan <- true
			case <-scheduler.stopChan:
				schedular.ticker.Stop()
				close(scheduler.stopChan)
			}
		}
	}()

	return nil
}

func (scheduler *Scheduler) Stop() {
	scheduler.stopChan <- true
}

func (scheduler *Scheduler) Wait() {
	<-scheduler.stopChan
}

func (scheduler *Scheduler) ScheduleAt(time time.Time, function Function, params ...FuncParam) error {
	funcID, err := schedular.FuncRegister.AddFunction(function, params...)
	if err != nil {
		return err
	}

	schedule := Schedule{
		IsRecurring: false,
		NextRun: time,
		LastRun: time.Unix(0, 0)
	}
	err = scheduler.Tasks.AddTask(schedule, funcID)
	if err != nil {
		return err
	}

	return nil
}

func (scheduler *Scheduler) ScheduleAfter(duration time.Duration, function Function, params ...FuncParam) error {
	return scheduler.ScheduleAt(time.Now().Add(duration), function, params...)
}

func (scheduler *Scheduler) ScheduleEvery(duration time.Duration, function Function, params ...FuncParam) error {
	funcID, err := schedular.FuncRegister.AddFunction(function, params...)
	if err != nil {
		return err
	}

	schedule := Schedule{
		IsRecurring: true,
		NextRun: time,
		LastRun: time.Unix(0, 0)
		Duration: duration
	}
	err = scheduler.Tasks.AddTask(schedule, funcID)
	if err != nil {
		return err
	}

	return nil
}

func (schedular *Schedular) runPending() {
	for id, task := range schedular.Tasks {
		task.IsDue() {
			if function, ok := schedular.FuncRegister[task.FunctionID]; ok {
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

