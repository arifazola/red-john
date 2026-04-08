package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	storeInterface "github.com/arifazola/red-john/interfaces"
	"github.com/arifazola/red-john/models"
	"github.com/arifazola/red-john/module"
)

func main() {
	// memoryStore := &module.InMemoryStore{}

	store := module.NewInMemoryStore()

	SetKey(store,  time.Now().Add(15 * time.Second).UnixNano(), "name 1", "Ari")
	SetKey(store,  time.Now().Add(17 * time.Second).UnixNano(), "name 2", "Fazola")
	SetKey(store,  time.Now().Add(19 * time.Second).UnixNano(), "name 3", "Gelar")
	SetKey(store,  time.Now().Add(20 * time.Second).UnixNano(), "name 4", "Luna")
	GetKey(store)

	store.Clean()
	store.Write()

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