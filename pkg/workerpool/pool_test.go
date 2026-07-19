package Taskerpool

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"
)

type testWorker struct {
	count atomic.Int32
}

func (worker *testWorker) PerformTask(Task) *Task {
	worker.count.Add(1)
	return nil
}

func TestPushRollsBackFullQueueMarker(t *testing.T) {
	queue := InitQueue(1, nil)
	if err := queue.Push(Task{ID: "first"}); err != nil {
		t.Fatal(err)
	}
	if err := queue.Push(Task{ID: "second"}); err == nil {
		t.Fatal("expected full queue error")
	}
	queue.Mux.RLock()
	_, marked := queue.InQueue["second"]
	queue.Mux.RUnlock()
	if marked {
		t.Fatal("failed task remained marked in queue")
	}
	queue.Stop()
}

func TestStopTerminatesWorkersAndRejectsTasks(t *testing.T) {
	queue := InitQueue(1, nil)
	worker := &testWorker{}
	queue.StartWorkers(worker)
	if err := queue.Push(Task{ID: "task"}); err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(time.Second)
	for worker.count.Load() == 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	queue.Stop()
	if err := queue.Push(Task{ID: "after-stop"}); err == nil {
		t.Fatal("expected stopped queue error")
	}
}

func TestQueueRecordsErrorsSafely(t *testing.T) {
	queue := InitQueue(1, nil)
	want := fmt.Errorf("task failed")
	queue.RecordError(want)
	errors := queue.Errors()
	if len(errors) != 1 || errors[0] != want {
		t.Fatalf("Errors() = %v, want [%v]", errors, want)
	}
	errors[0] = nil
	if queue.Errors()[0] != want {
		t.Fatal("Errors returned internal queue storage")
	}
}
