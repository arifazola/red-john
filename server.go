package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"strings"

	"github.com/arifazola/red-john/module"
)

type Server struct {
	inMemoryStore *module.InMemoryStore
}

func(server *Server) StartServer(context context.Context) {
	ln, err := net.Listen("tcp", ":8080")

	if err != nil {
		fmt.Println("error listening network ", err)
	}

	go func ()  {
		<-context.Done()
		ln.Close()
	}()

	for {
		conn, err := ln.Accept()

		if err != nil {
			fmt.Println("error connection ", err)
			return
		}

		fmt.Println("connected")

		go server.handleConnection(conn)
	}
}

func(server *Server) handleConnection(connection net.Conn) {

	defer connection.Close()

	reader := bufio.NewReader(connection)

	for {
		msg, err := reader.ReadString('\n')

		if err != nil {
			if err == io.EOF {
				fmt.Println("client disconnected")
			} else {
				fmt.Println("read error:", err)
			}

			return
		}

		msg = strings.TrimSpace(msg)
		if msg == ""{continue}

		commands := module.TextTokenizer(msg)

		commandResult, err := module.CommandRouter(commands, server.inMemoryStore)

		if err != nil {
			fmt.Println("Command error", err)
			connection.Write([]byte("ERR " + err.Error() + "\n"))
			// return
		} else {
			connection.Write([]byte(commandResult))
		}
	}

}