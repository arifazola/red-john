package module

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"sync"
	"time"

	"github.com/arifazola/red-john/models"
)

type InMemoryStore struct {
	mut  sync.RWMutex
	data map[string]models.Item
}

func NewInMemoryStore() *InMemoryStore {
	newInMemoryStore := &InMemoryStore{
		data: make(map[string]models.Item),
	}
	
	return newInMemoryStore
}

func (s *InMemoryStore) Get(key string) (models.Item, bool) {
	s.mut.RLock()
	defer s.mut.RUnlock()
	val, ok := s.data[key]
	return val, ok
}

func (s *InMemoryStore) Set(key string, value models.Item) {
	s.mut.Lock()
	defer s.mut.Unlock()
	s.data[key] = value
}

func (s *InMemoryStore) Delete (key string) {
	s.mut.Lock()
	defer s.mut.Unlock()
	delete(s.data, key)
}

func (s *InMemoryStore) GetAll() (map[string]models.Item, bool){
	s.mut.RLock()
	defer s.mut.RUnlock()
	copyData := make(map[string]models.Item, len(s.data))
	maps.Copy(copyData, s.data)
	return copyData, true
}

func (s *InMemoryStore) Clean(context context.Context){
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-context.Done():
			fmt.Println("Cleaner stopping")
			return
		case <-ticker.C:
			s.deleteExpired()
		}
	}
}

func (s *InMemoryStore) Write(context context.Context){
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select{
		case <-context.Done():
			fmt.Println("Writer stopping")
			return
		case <-ticker.C:
			s.WriteToJson()
		}
	}
}

func (s *InMemoryStore) deleteExpired(){
	now := time.Now().UnixNano()
	s.mut.Lock()
	defer s.mut.Unlock()

	for key, value := range s.data {
		if(value.ExpiresAt > 0 && now > value.ExpiresAt){
			fmt.Println("Deleting ", value.Value)
			delete(s.data, key)
		}
	}
}

func (s *InMemoryStore) WriteToJson(){
	s.mut.RLock()
    tempCopy := make(map[string]models.Item, len(s.data))
    maps.Copy(tempCopy, s.data)
    s.mut.RUnlock()

	tempPath := "data.json.tmp"
	finalPath := "data.json"

	file, err := os.Create(tempPath)
	
	if err != nil {
		println("Error while opening file")
		return
	}

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", " ")

	if err := encoder.Encode(tempCopy); err != nil {
		file.Close()
		fmt.Println("error encoding json ", err)
		return
	}

	if err := file.Sync(); err != nil {
		file.Close()
		println("error syncing ", err)
		return
	}

	file.Close()

	if err := os.Rename(tempPath, finalPath); err != nil {
		fmt.Println("error during atomic rename ", err)
	}
}