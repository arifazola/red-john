package main

import (
	"fmt"
	"net"
)

func ConnectToLeader(leaderAddr string) {
	conn, err := net.Dial("tcp", ":"+leaderAddr)

	if err != nil {
		fmt.Println("Error connecting to leader server ", err)
		return
	}

	message := "Hello from follower\n"

	_, writeError := conn.Write([]byte(message))

	if writeError != nil {
		fmt.Println("error writing message ", writeError)
		return
	}

}