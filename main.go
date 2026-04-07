package main

import (
	"fmt"

	storeInterface "github.com/arifazola/red-john/interfaces"
	inMemoryStore "github.com/arifazola/red-john/module"
)

func main() {
	m := make(map[string]string)
	memoryStore := &inMemoryStore.InMemoryStore{
		Data: m,
	}
	
	SetKey(memoryStore)
	GetKey(memoryStore)

}

func GetKey(store storeInterface.Store){
	val, _ := store.Get("name")
	fmt.Println(val)
}

func SetKey(store storeInterface.Store){
	store.Set("name", "fazola")
}