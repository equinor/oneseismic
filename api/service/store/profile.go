package store

import (
	"fmt"
	"sync"
)

type ProfileStore interface {
	Append(string, map[string]string) error
	Fetch(string) (map[string]string, error)
}

type ProfileInMemoryStore struct {
	lock  sync.RWMutex
	store map[string]map[string]string
}

func (ps *ProfileInMemoryStore) Append(sessionID string, data map[string]string) error {

	ps.lock.Lock()
	defer ps.lock.Unlock()

	if ps.store == nil {
		ps.store = make(map[string]map[string]string, 0)
	}
	r, ok := ps.store[sessionID]

	if !ok {
		r = make(map[string]string, 0)
	}
	for k, v := range data {
		r[k] = v
	}
	ps.store[sessionID] = r

	return nil
}

func (ps *ProfileInMemoryStore) Fetch(sessionID string) (map[string]string, error) {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	r, ok := ps.store[sessionID]

	if !ok {
		return nil, fmt.Errorf("No profile matching session %s", sessionID)
	}

	return r, nil
}
