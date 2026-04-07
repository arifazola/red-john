package main_test

import (
	"testing"
)

func TestRace(t *testing.T) {
	// m := make(map[string]string)
	// memoryStore := inMemoryStore.InMemoryStore{
	// 	Data: m,
	// }

	// var wg sync.WaitGroup
	// wg.Add(3)

	// go func ()  {
	// 	memoryStore.Set("name", "Ari")
	// 	defer wg.Done()
	// }()

	// go func ()  {
	// 	memoryStore.Set("name", "Fazoladffd")
	// 	defer wg.Done()
	// }()

	// go func ()  {
	// 	time.Sleep(3 * time.Second)
	// 	val, _ := memoryStore.Get("name")
	// 	fmt.Println(val)
	// 	defer wg.Done()
	// }()

	// wg.Wait()
}