package utils

import (
	"sync"
)

type locker struct {
	lock  *sync.Mutex
	count int
}

type SyncLocker struct {
	lockerList map[string]*locker
	mapLocker  *sync.Mutex
}

func NewSyncLocker() *SyncLocker {
	lock := &SyncLocker{
		mapLocker: &sync.Mutex{},
	}
	lock.lockerList = make(map[string]*locker)

	return lock
}

func (l *SyncLocker) Lock(key string) {
	var tmp *locker
	l.mapLocker.Lock()
	if l.lockerList[key] == nil {
		tmp = &locker{
			lock:  &sync.Mutex{},
			count: 0,
		}
		l.lockerList[key] = tmp
	} else {
		tmp = l.lockerList[key]
	}
	l.lockerList[key].count++
	l.mapLocker.Unlock()
	tmp.lock.Lock()
}

func (l *SyncLocker) Unlock(key string) {
	if l.lockerList[key] == nil {
		return
	}
	var tmp *locker
	l.mapLocker.Lock()
	tmp = l.lockerList[key]
	l.lockerList[key].count--
	if l.lockerList[key].count == 0 {
		delete(l.lockerList, key)
	}
	l.mapLocker.Unlock()
	tmp.lock.Unlock()
}
