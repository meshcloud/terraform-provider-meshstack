package xsync

import "sync"

type KeyedMutex struct {
	mu    sync.Mutex
	locks map[string]*sync.Mutex
}

func (m *KeyedMutex) Lock(key string) (unlock func()) {
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
