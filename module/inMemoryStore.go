package module

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/arifazola/red-john/models"
)

type InMemoryStore struct {
	mut  sync.RWMutex
	data map[string]models.Item
}

func (s *InMemoryStore) NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		data: make(map[string]models.Item),
	}
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
	
}

func (s *InMemoryStore) GetAll() (map[string]models.Item, bool){
	s.mut.RLock()
	defer s.mut.RUnlock()
	return s.data, true
}

func (s *InMemoryStore) Clean(){

	ticker := time.NewTicker(1 * time.Second)

	go func ()  {
		for range ticker.C{
			fmt.Println("run cleaner")
			s.deleteExpired()
		}
	}()

}

func (s *InMemoryStore) Write(){
	ticker := time.NewTicker(10 * time.Second)

	go func ()  {
		for range ticker.C{
			fmt.Println("writing data to disk")
			s.writeToJson()
		}
	}()
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

func (s *InMemoryStore) writeToJson(){
	s.mut.RLock()
	defer s.mut.RUnlock()
	file, err := os.Create("data.json")
	
	if err != nil {
		panic (err)
	}
	
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", " ")

	if err := encoder.Encode(s.data); err != nil {
		panic(err)
	}
}