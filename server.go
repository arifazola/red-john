package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/arifazola/red-john/enums"
	"github.com/arifazola/red-john/models"
	"github.com/arifazola/red-john/module"
)

type Server struct {
	inMemoryStore *module.InMemoryStore
	Addr, LeaderAddr, Role  string
	followerMut sync.Mutex
	followers []net.Conn
}

func(server *Server) StartServer(context context.Context) {
	ln, err := net.Listen("tcp", ":"+server.Addr)

	if err != nil {
		fmt.Println("error listening network ", err)
		return
	}

	go func ()  {
		<-context.Done()
		ln.Close()
	}()

	if err := server.SyncLocalData(); err != nil {
		log.Fatalf("Critical error loading local data: %v", err)
	}
	

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

func(server *Server) SyncLocalData() error{
	data, err := os.ReadFile("data.json")

	if err != nil {
		if os.IsNotExist(err){
			fmt.Println("No existing data. Starting fresh")
			return nil
		}

		return fmt.Errorf("failed to read data file: %w", err)
	}

	var parsedJson map[string]models.Item

	jsonErr := json.Unmarshal(data, &parsedJson)

	if jsonErr != nil {
		return fmt.Errorf("Failed to parse data. File might be corrupted: %w", err)
	}

	server.inMemoryStore.SetAll(parsedJson)

	return nil
}

func(server *Server) handleConnection(connection net.Conn) {
	shouldCloseConnection := true //flag
	defer func ()  {
		if shouldCloseConnection{
			fmt.Println("CLOSING CONNECTION")
			connection.Close()
		} else {
			fmt.Println("Connection stays opened")
		}
	}()

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

		fmt.Println("message ", msg)

		msg = strings.TrimSpace(msg)
		if msg == ""{continue}

		if(msg == "SYNC_ME"){
			// connection.Write([]byte("YOU ARE SYNCED\n"))
			fmt.Println("Sending data to follower")
			server.followerMut.Lock()
			defer server.followerMut.Unlock()
			server.SendSnapshotToFollower(connection)
			server.followers = append(server.followers, connection)
			shouldCloseConnection = false
			return;
		}

		commands := module.TextTokenizer(msg)

		if server.Role == enums.RoleLeader && commands[0] == "SET" {
			fmt.Println("Broadcasting SET command to followers")
			server.BroadcastToFollowers(msg)
		}

		commandResult, err := module.CommandRouter(commands, server.inMemoryStore, server.Role)

		if err != nil {
			fmt.Println("Command error", err)
			connection.Write([]byte("ERR " + err.Error() + "\n"))
			// return
		} else {
			connection.Write([]byte(commandResult))
		}
	}

}

func(server *Server) SendSnapshotToFollower(conn net.Conn) error {
	fmt.Println("Serializing In memory data")
	data, err := server.serializeInMemoryData()

	if err != nil {
		fmt.Println("serialize error ", err)
		return err
	}
	fmt.Println("leader data", data)
	conn.Write([]byte(data + "\n"))

	return nil
}

func(server *Server) serializeInMemoryData() (string, error){
	fmt.Println("fdsf")
	server.inMemoryStore.Mut.Lock()
	defer server.inMemoryStore.Mut.Unlock()

	data := server.inMemoryStore.GetAllUnsafe()

	json, err := json.Marshal(data)

	if err != nil {
		fmt.Println("error json marshal ", err)
		return "", err
	}

	fmt.Println("json result ")
	fmt.Println(string(json))

	return string(json), nil 
}

func(server *Server) BroadcastToFollowers(command string){
	server.followerMut.Lock()
	defer server.followerMut.Unlock()

	fmt.Println("Followers list", server.followers)

	var activeFollowers []net.Conn

	for _, conn := range server.followers {
		_, err := conn.Write([]byte(command + "\n"))

		if err != nil {
			fmt.Println("Failed to send command to follower ", err)
			continue
		}

		activeFollowers = append(activeFollowers, conn)
	}

	server.followers = activeFollowers
}

func (server *Server) SendHeartbeat(conn net.Conn, context context.Context){
	ticker := time.NewTicker(30 * time.Second)

	for{
		select{
		case <-context.Done():
			fmt.Println("Stopping heartbeat")
			return
		case <-ticker.C:
			server.BroadcastToFollowers("")
		}
	}
	
}