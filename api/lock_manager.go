package api

import (
	"sync"
	log "github.com/sirupsen/logrus"
)

var (
    lockManagerInstance *UserLockManager
    once                sync.Once
)

type UserLockManager struct {
    locks sync.Map
}

func GetUserLockManager() *UserLockManager {
    once.Do(func() {
        lockManagerInstance = &UserLockManager{}
    })
    return lockManagerInstance
}

func (m *UserLockManager) getLock(lockId string) *sync.Mutex {
    lock, _ := m.locks.LoadOrStore(lockId, &sync.Mutex{})
    return lock.(*sync.Mutex)
}

func (m *UserLockManager) TryLock(lockId string) bool {
    lock := m.getLock(lockId)
    return lock.TryLock()
}

func (m *UserLockManager) Unlock(userID string) {
    lockVal, exists := m.locks.Load(userID)
    if !exists {
        log.Errorf("unlock called for non-existent lock, user_id: %s", userID)
        return
    }
    
    lock := lockVal.(*sync.Mutex)
    
    // Catch panic from double-unlock
    defer func() {
        if r := recover(); r != nil {
            log.Errorf("double unlock detected for user_id: %s", userID)
        }
    }()
    
    lock.Unlock()
}

// Run a scheduler for periodically cleanup locks for inactive users
func (m *UserLockManager) Cleanup() {
    m.locks.Range(func(key, value interface{}) bool {
        mutex := value.(*sync.Mutex)
        // Only delete if mutex is unlocked (not in use)
        if mutex.TryLock() {
            mutex.Unlock()
            m.locks.Delete(key)
        }
        return true
    })
}
