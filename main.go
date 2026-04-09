package main

import (
	"context"
	"flag"
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
	port := flag.String("port", "8080", "The port the server will listen to")
	leaderAddr := flag.String("leader", "", "Address of the leader server (if this is a follower)")

	flag.Parse()

	role := "LEADER"

	if *leaderAddr != "" {
		role = "FOLLOWER"
	}

	fmt.Printf("Starting server as %s on port %s\n", role, *port)
    if role == "FOLLOWER" {
        fmt.Printf("Connecting to leader at %s\n", *leaderAddr)
    }



	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	memoryStore := module.NewInMemoryStore()
	
	server := Server{
		inMemoryStore: memoryStore,
		Addr: *port,
		LeaderAddr: *leaderAddr,
		Role: role,
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