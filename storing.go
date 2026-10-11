package main

import (
	"sync"
	"time"
)

type Entry struct {
	value []byte
	expiresAt time.Time
}

// expireTime should be -1 if it doesnt have an expiration date
func makeEntry(value []byte, expireTime time.Duration) Entry {
	if expireTime == -1 {
		return Entry{value: value}
	}
	return Entry{value: value, expiresAt: time.Now().UTC().Add(expireTime)}
}

func (e Entry) expired() bool {
	return !e.expiresAt.IsZero() && !time.Now().UTC().Before(e.expiresAt)
}

type Store struct {
	data map[string]Entry
	mx sync.Mutex
}

func NewStore() Store {
	return Store{ data: make(map[string]Entry) }
}

// returns TRUE if the element should still exist and FALSE if it expired and got deleted
func (s *Store) getLive(key string) (Entry, bool) {
	s.mx.Lock()
	defer s.mx.Unlock()

	entry, ok := s.data[key]
	if !ok {
		return Entry{}, false
	}

	if entry.expired() {
		delete(s.data, key)
		return Entry{}, false
	}
	return entry, true
}

func (s *Store) Set(key string, value []byte) {
	s.mx.Lock()
	defer s.mx.Unlock()

	s.data[key] = makeEntry(value, -1)
}

func (s *Store) Get(key string) ([]byte, bool) {
	entry, ok := s.getLive(key)

	return entry.value, ok
}

func (s *Store) Exists(key string) bool {
	_, ok := s.getLive(key)

	return ok
}

func (s *Store) Del(key string) bool {
	s.mx.Lock()
	defer s.mx.Unlock()

	entry, ok := s.data[key]
	if !ok {
		return false
	}
	delete(s.data, key)
	return !entry.expired() // if its expired it "was already" deleted
}

func (s *Store) Expire(key string, expireAfter time.Duration) bool {
	s.mx.Lock()
	defer s.mx.Unlock()

	entry, ok := s.data[key]
	if !ok {
		return false
	}
	if entry.expired() {
		delete(s.data, key)
		return false
	}

	entry.expiresAt = time.Now().UTC().Add(expireAfter)
	s.data[key] = entry

	return true
}

// returns time to expiration in seconds; if it doesnt have and expiration date, returns -1; if the key doesnt exists, returns -2 
func (s *Store) Ttl(key string) int {
	entry, ok := s.getLive(key)
	if !ok {
		return -2
	}

	if entry.expiresAt.IsZero() {
		return -1
	}

	return int(entry.expiresAt.Unix() - time.Now().UTC().Unix())
}
