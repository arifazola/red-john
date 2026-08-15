package main

import (
	"context"
	"flag"
	"fmt"
	"math/rand/v2"
	"os"
	"os/signal"
	"sync"
	"syscall"

	storeInterface "github.com/arifazola/red-john/interfaces"
	"github.com/arifazola/red-john/models"
	"github.com/arifazola/red-john/module"
	"github.com/google/uuid"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	port := flag.String("port", "8080", "The port the server will listen to")
	leaderAddr := flag.String("leader", "", "Address of the leader server (if this is a follower)")

	flag.Parse()

	role := "LEADER"

	if *leaderAddr != "" {
		role = "FOLLOWER"
	}

	memoryStore := module.NewInMemoryStore()

	var wg sync.WaitGroup

	fmt.Printf("Starting server as %s on port %s\n", role, *port)
	min, max := 1, 1000
	rangeNum := rand.IntN(max-min+1) + min
	raftData := models.RaftData{
		ElectionInterval: rangeNum,
		Role: role,
	}
	serverID := uuid.New().String()
	server := Server{
		inMemoryStore: memoryStore,
		RaftData: &raftData,
		Addr: *port,
		LeaderAddr: *leaderAddr,
		Role: role,
		ServerID: serverID,
	}

	wg.Add(3)

	go func ()  {
		defer wg.Done()
		server.StartServer(ctx)
	}()

    if role == "FOLLOWER" {
		wg.Add(1)
        fmt.Printf("Connecting to leader at %s\n", *leaderAddr)
		
		go func ()  {
			defer wg.Done()
			server.ConnectToLeader(*leaderAddr, ctx)
		}()
    }

	defer stop()
	
	// server := Server{
	// 	inMemoryStore: memoryStore,
	// 	Addr: *port,
	// 	LeaderAddr: *leaderAddr,
	// 	Role: role,
	// }

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