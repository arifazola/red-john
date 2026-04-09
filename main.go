package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	storeInterface "github.com/arifazola/red-john/interfaces"
	"github.com/arifazola/red-john/models"
	"github.com/arifazola/red-john/module"
)

func main() {
	memoryStore := module.NewInMemoryStore()
	
	server := Server{
		inMemoryStore: memoryStore,
	}
	go server.StartServer()
	go memoryStore.Clean()
	go memoryStore.Write()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	fmt.Println("Store is running. Press ctrl + c to stop")

	<- sigChan
	fmt.Println("Shutting down gracefully")

}

func GetKey(store storeInterface.Store){
	val, _ := store.Get("name")
	fmt.Println(val)
}

func SetKey(store storeInterface.Store, expires int64, key, value string){
	item := models.Item{
		Value: value,
		ExpiresAt: expires,
	}
	store.Set(key, item)
}