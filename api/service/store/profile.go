package store

import (
	"fmt"
	"sync"
	"time"
)

type ProfileStore interface {
	Append(string, map[string]string) error
	Fetch(string) (map[string]string, error)
}

type cacheItem struct {
	Data     map[string]string
	Modified time.Time
}

type ProfileInMemoryStore struct {
	lock  sync.RWMutex
	store map[string]cacheItem
}

func (ps *ProfileInMemoryStore) Append(sessionID string, data map[string]string) error {

	ps.lock.Lock()
	defer ps.lock.Unlock()

	if ps.store == nil {
		ps.store = make(map[string]cacheItem, 0)
	}
	r, ok := ps.store[sessionID]

	if !ok {
		r = cacheItem{Data: make(map[string]string, 0)}
	}
	for k, v := range data {
		r.Data[k] = v
		r.Modified = time.Now()
	}
	ps.store[sessionID] = r

	purgeList := make([]string, 0)
	for k, v := range ps.store {
		if v.Modified.Sub(time.Now()) > time.Hour {
			purgeList = append(purgeList, k)
		}
	}
	for _, v := range purgeList {
		delete(ps.store, v)
	}
	return nil
}

func (ps *ProfileInMemoryStore) Fetch(sessionID string) (map[string]string, error) {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	r, ok := ps.store[sessionID]

	if !ok {
		return nil, fmt.Errorf("No profile matching session %s", sessionID)
	}

	return r.Data, nil
}
