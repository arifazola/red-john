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
	Mut  sync.RWMutex
	data map[string]models.Item
}

func NewInMemoryStore() *InMemoryStore {
	newInMemoryStore := &InMemoryStore{
		data: make(map[string]models.Item),
	}
	
	return newInMemoryStore
}

func (s *InMemoryStore) Get(key string) (models.Item, bool) {
	s.Mut.RLock()
	defer s.Mut.RUnlock()
	val, ok := s.data[key]
	return val, ok
}

func (s *InMemoryStore) Set(key string, value models.Item) {
	s.Mut.Lock()
	defer s.Mut.Unlock()
	s.data[key] = value
}

func (s *InMemoryStore) Delete (key string) {
	s.Mut.Lock()
	defer s.Mut.Unlock()
	delete(s.data, key)
}

func (s *InMemoryStore) GetAll() (map[string]models.Item, bool){
	s.Mut.RLock()
	defer s.Mut.RUnlock()
	copyData := make(map[string]models.Item, len(s.data))
	maps.Copy(copyData, s.data)
	return copyData, true
}

func (s *InMemoryStore) GetAllUnsafe() map[string]models.Item {
	return s.data
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

func (s *InMemoryStore) SetAll(data map[string]models.Item){
	s.Mut.Lock()
	defer s.Mut.Unlock()
	s.data = data
}

func (s *InMemoryStore) Write(context context.Context){
	ticker := time.NewTicker(30 * time.Second)
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
	s.Mut.Lock()
	defer s.Mut.Unlock()

	for key, value := range s.data {
		if(value.ExpiresAt > 0 && now > value.ExpiresAt){
			fmt.Println("Deleting ", value.Value)
			delete(s.data, key)
		}
	}
}

func (s *InMemoryStore) WriteToJson(){
	s.Mut.RLock()
    tempCopy := make(map[string]models.Item, len(s.data))
    maps.Copy(tempCopy, s.data)
    s.Mut.RUnlock()

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

	ClearLog()
}