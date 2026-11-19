package provider

import "sync"

type keyedMutex struct {
	mu    sync.Mutex
	locks map[string]*sync.Mutex
}

func (m *keyedMutex) lock(key string) (unlock func()) {
	m.mu.Lock()
	if m.locks == nil {
		m.locks = map[string]*sync.Mutex{}
	}
	l, ok := m.locks[key]
	if !ok {
		l = &sync.Mutex{}
		m.locks[key] = l
	}
	m.mu.Unlock()

	l.Lock()
	return l.Unlock
}

var meshObjectLocks keyedMutex

func lockWorkspace(name string) (unlock func()) {
	return meshObjectLocks.lock("meshWorkspace/" + name)
}

func lockProject(workspace, name string) (unlock func()) {
	return meshObjectLocks.lock("meshProject/" + workspace + "." + name)
}
