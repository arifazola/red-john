package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strings"

	"github.com/arifazola/red-john/models"
	"github.com/arifazola/red-john/module"
)

type Client struct {
	inMemoryStore *module.InMemoryStore
}

func(client *Client) ConnectToLeader(leaderAddr string, context context.Context) {
	conn, err := net.Dial("tcp", leaderAddr)

	if err != nil {
		fmt.Println("Error connecting to leader server ", err)
		return
	}

	go func ()  {
		<-context.Done()
		conn.Close()
	}()

	defer conn.Close()

	reader := bufio.NewReader(conn)

	_, writeError := conn.Write([]byte("SYNC_ME\n"))

	if writeError != nil {
		fmt.Println("error writing message ", writeError)
		return
	}

	for {
		msg, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Lost connection to leader", err)
			return
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