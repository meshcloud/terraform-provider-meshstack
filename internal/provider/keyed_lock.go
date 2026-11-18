package provider

import "sync"

type keyedLock struct {
	locks sync.Map
}

func (l *keyedLock) lock(key string) (unlock func()) {
	value, _ := l.locks.LoadOrStore(key, &sync.Mutex{})
	mutex, _ := value.(*sync.Mutex)
	mutex.Lock()
	return mutex.Unlock
}

var workspaceUserGroupWrites keyedLock
