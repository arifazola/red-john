package main

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"strings"

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


	_, writeError := conn.Write([]byte("SYNC_ME\n"))

	if writeError != nil {
		fmt.Println("error writing message ", writeError)
		return
	}

	go func ()  {
		defer conn.Close()

		reader := bufio.NewReader(conn)

		for {
			select {
			case <-context.Done():
				fmt.Println("follower sync stopping")
			default :
				
			}

			msg, err := reader.ReadString('\n')
				if err != nil {
					fmt.Println("Lost connection to leader", err)
					return
				}

				fmt.Println("Command from leader")
				commands := strings.Fields(msg)

				fmt.Println("Adding command for followers")
				result, err := module.CommandRouter(commands, client.inMemoryStore, "")

				fmt.Println("client command result ", result)
		}
	}()

}