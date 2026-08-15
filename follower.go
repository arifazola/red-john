package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net"
	"strings"
	"time"

	"github.com/arifazola/red-john/models"
	"github.com/arifazola/red-john/module"
)

type Follower struct {
	inMemoryStore *module.InMemoryStore
	RaftData *models.RaftData
	timer *time.Timer
}

func(client *Follower) ConnectToLeader(leaderAddr string, context context.Context) {
	maxNumOfRetry := 5
	
	for i := 1; i < maxNumOfRetry; i ++ {
		conn, err := net.Dial("tcp", leaderAddr)

		if err != nil {
			fmt.Println("Error connecting to leader server ", err)

			select {
			case <-time.After(2 * time.Second):
				continue
			case <-context.Done():
				fmt.Println("Shutting down signal received")
				return
			}
		}


		reader := bufio.NewReader(conn)

		_, writeError := conn.Write([]byte("SYNC_ME\n"))

		if writeError != nil {
			fmt.Println("error writing message ", writeError)
			conn.Close()
			continue // continue to the next loop to retry 
		}

		go func ()  {
			client.StartElectionTimer()
		}()

		for {
			conn.SetReadDeadline(time.Now().Add(5 * time.Second))
			msg, err := reader.ReadString('\n')
			if err != nil {
				fmt.Println("Lost connection to leader", err)
				conn.Close()
				i = maxNumOfRetry
				break //break to get out of this inner loop and retry
			}

			fmt.Println("Command from leader", msg)

			var parsedJson map[string]models.Item
			jsonErr := json.Unmarshal([]byte(msg), &parsedJson)

			//check if parsing the message from leader throws an error
			//if message cannot be parsed, then it's a command (SET or GET)
			//if message can be parsed, then it's a syncing data. Follower must store data from leader to it's own inMemoryStore
			if jsonErr != nil {
				commands := strings.Fields(msg)
				
				if commands[0] == "PING"{
					client.ResetElectionTimer()
					conn.Write([]byte("PONG\n"))
					continue
				}

				fmt.Println("Adding command for followers", commands)
				result, _ := module.CommandRouter(commands, client.inMemoryStore, "")

				fmt.Println("client command result ", result)
				conn.Write([]byte("STORED\n"))
			} else {
				fmt.Println("Syncing")
				client.inMemoryStore.SetAll(parsedJson)
				fmt.Println("Finished syncing data")
			}
		}
	}

}

func (client *Follower) ResetElectionTimer(){
	fmt.Println(">>> RESET ELECTION TIMER")
	client.timer.Stop()
	min, max := 4000, 10000
	rangeNum := rand.IntN(max-min+1) + min
	client.timer.Reset(time.Duration(rangeNum) * time.Millisecond)
}

func (client *Follower) StartElectionTimer(){
	fmt.Println(">>> StartElectionTimer CALLED")
	min, max := 4000, 10000
	rangeNum := rand.IntN(max-min+1) + min
	client.RaftData.ElectionInterval = rangeNum 
	client.timer = time.NewTimer(time.Duration(client.RaftData.ElectionInterval) * time.Millisecond)
	
	for {
		fmt.Println("Election timer start with interval:", client.RaftData.ElectionInterval)

		<-client.timer.C
		
		fmt.Println("Should start election")
		//Start election implementation
	}
}