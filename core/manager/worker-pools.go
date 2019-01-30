package manager

import (
	"fmt"
	"runtime"
	"sync"

	"github.com/sdslabs/beastv4/core"
	log "github.com/sirupsen/logrus"
)

type Queue struct {
	WorkQueue chan Work
	Mux       sync.RWMutex
	Set       map[string]bool
}

var Q *Queue

type DeployInfo struct {
	ChallDir   string
	SkipStage  bool
	SkipCommit bool
	Purge      bool
}

type Work struct {
	Action    string
	ChallName string
	Info      DeployInfo
}

func (q *Queue) CheckPush(w Work) error {
	q.Mux.Lock()
	if _, ex := q.Set[w.ChallName]; ex {
		q.Mux.Unlock()
		return fmt.Errorf("The challenge : %s is already in queue", w.ChallName)
	}
	q.Set[w.ChallName] = true
	q.Mux.Unlock()
	q.WorkQueue <- w
	// TODO : get size of the queue
	return nil
}

func (q *Queue) Remove(challName string) {
	q.Mux.Lock()
	delete(q.Set, challName)
	q.Mux.Unlock()
}

func startConcurrentWorker(i int) {
	for {
		w := <-Q.WorkQueue
		switch w.Action {
		case core.MANAGE_ACTION_DEPLOY:
			StartDeployPipeline(w.Info.ChallDir, w.Info.SkipStage, w.Info.SkipCommit)
			Q.Remove(w.ChallName)

		case core.MANAGE_ACTION_UNDEPLOY:
			err := StartUndeployChallenge(w.ChallName, false)
			if err != nil {
				log.Errorf("Error while undeplying challenge(%s): %s", w.ChallName, err.Error())
			}
			Q.Remove(w.ChallName)

		case core.MANAGE_ACTION_REDEPLOY:
			err := StartUndeployChallenge(w.ChallName, true)
			Q.Remove(w.ChallName)
			if err != nil {
				log.Errorf("Error while redeplying challenge(%s): %s", w.ChallName, err.Error())
				continue
			}
			err = DeployChallenge(w.ChallName)
			if err != nil {
				log.Error(err)
			}

		case core.MANAGE_ACTION_PURGE:
			err := StartUndeployChallenge(w.ChallName, true)
			if err != nil {
				log.Errorf("Error while purging challenge(%s): %s", w.ChallName, err.Error())
			}
			Q.Remove(w.ChallName)

		default:
			log.Errorf("The action(%s) specified for challenge : %s does not exist", w.Action, w.ChallName)
			Q.Remove(w.ChallName)
		}
	}
}

func StartWorkers() {
	numCPUs := runtime.NumCPU()
	log.Info("Total workers: ", numCPUs)
	for i := 0; i < numCPUs; i++ {
		go startConcurrentWorker(i)
	}
}

func InitQueue() {
	Q = &Queue{
		WorkQueue: make(chan Work, core.MAX_QUEUE_SIZE),
		Mux:       sync.RWMutex{},
		Set:       map[string]bool{},
	}
}
