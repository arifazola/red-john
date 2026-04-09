package main

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"strings"
)

func ConnectToLeader(leaderAddr string, context context.Context) {
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
				msg, err := reader.ReadString('\n')
				if err != nil {
					fmt.Println("Lost connection to leader", err)
					return
				}

				fmt.Println("Command from leader")
				commands := strings.Fields(msg)

				fmt.Println(commands)
			}
		}
	}()

}