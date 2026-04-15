package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/arifazola/red-john/models"
	"github.com/arifazola/red-john/module"
)

type Client struct {
	inMemoryStore *module.InMemoryStore
}

func(client *Client) ConnectToLeader(leaderAddr string, context context.Context) {
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

		for {
			conn.SetReadDeadline(time.Now().Add(5 * time.Second))
			msg, err := reader.ReadString('\n')
			if err != nil {
				netErr, ok := err.(net.Error)
				if ok && netErr.Timeout(){
					select{
					case <-context.Done():
						fmt.Println("Shutting down reading")
						conn.Close()
						return
					default:
						fmt.Println("No message received after 5 second. Timeout")
						continue
					}
				}
				fmt.Println("Lost connection to leader", err)
				conn.Close()
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

				fmt.Println("Adding command for followers", commands)
				result, _ := module.CommandRouter(commands, client.inMemoryStore, "")

				fmt.Println("client command result ", result)
			} else {
				fmt.Println("Syncing")
				client.inMemoryStore.SetAll(parsedJson)
				fmt.Println("Finished syncing data")
			}
		}
	}

}