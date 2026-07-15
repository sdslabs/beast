package scheduler

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestSchedulerRejectsNonPositiveRecurringDuration(t *testing.T) {
	scheduler := newScheduler(time.Millisecond)
	if err := scheduler.ScheduleEvery(0, func() {}); err == nil {
		t.Fatal("expected invalid duration error")
	}
	scheduler.Stop()
}

func TestSchedulerDoesNotOverlapRecurringTask(t *testing.T) {
	scheduler := newScheduler(time.Millisecond)
	var running atomic.Int32
	var maximum atomic.Int32
	task := func() {
		current := running.Add(1)
		for {
			old := maximum.Load()
			if current <= old || maximum.CompareAndSwap(old, current) {
				break
			}
		}
		time.Sleep(10 * time.Millisecond)
		running.Add(-1)
	}
	if err := scheduler.ScheduleEvery(time.Millisecond, task); err != nil {
		t.Fatal(err)
	}
	scheduler.Start()
	time.Sleep(30 * time.Millisecond)
	scheduler.Stop()
	if maximum.Load() != 1 {
		t.Fatalf("maximum concurrent executions = %d, want 1", maximum.Load())
	}
}

func TestSchedulerSupportsConcurrentScheduling(t *testing.T) {
	scheduler := newScheduler(time.Millisecond)
	scheduler.Start()
	var group sync.WaitGroup
	for i := 0; i < 20; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			if err := scheduler.ScheduleAfter(time.Millisecond, func() {}); err != nil {
				t.Errorf("schedule task: %v", err)
			}
		}()
	}
	group.Wait()
	time.Sleep(5 * time.Millisecond)
	scheduler.Stop()
}
