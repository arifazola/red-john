package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
)

func StartServer() {
	ln, err := net.Listen("tcp", ":8080")

	if err != nil {
		fmt.Println("error listening network ", err)
	}

	defer ln.Close()

	for {
		conn, err := ln.Accept()

		if err != nil {
			fmt.Println("error connection ", err)
			continue
		}

		fmt.Println("connected")

		go handleConnection(conn)
	}
}

func handleConnection(connection net.Conn) {
	// defer connection.Close()

	// buffer := make([]byte, 1024)

	// for {
	// 	bufferData, err := connection.Read(buffer)

	// 	if err != nil {
	// 		fmt.Println("error read", err)
	// 	}

	// 	msg := string(buffer[:bufferData])

	// 	println(msg)
	// }

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

		println(msg)
	}

}