package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	storeInterface "github.com/arifazola/red-john/interfaces"
	"github.com/arifazola/red-john/models"
	"github.com/arifazola/red-john/module"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	memoryStore := module.NewInMemoryStore()
	
	server := Server{
		inMemoryStore: memoryStore,
	}

	var wg sync.WaitGroup

	wg.Add(3)

	go func ()  {
		defer wg.Done()
		server.StartServer(ctx)
	}()

	go func ()  {
		defer wg.Done()
		memoryStore.Clean(ctx)
	}()

	go func ()  {
		defer wg.Done()
		memoryStore.Write(ctx)
	}()

	<-ctx.Done()
	
	fmt.Println("Saving data")

	memoryStore.WriteToJson()

	wg.Wait()

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