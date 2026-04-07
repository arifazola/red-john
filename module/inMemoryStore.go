package inMemoryStore

import "sync"

type InMemoryStore struct {
	mut  sync.RWMutex
	Data map[string]string
}


func (s *InMemoryStore) Get(key string) (string, bool) {
	s.mut.RLock()
	defer s.mut.RUnlock()
	val, ok := s.Data[key]
	return val, ok
}

func (s *InMemoryStore) Set(key, value string) {
	s.mut.Lock()
	defer s.mut.Unlock()
	s.Data[key] = value
}

func (s *InMemoryStore) Delete (key string) {
	
}