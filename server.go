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
	"strconv"
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
	followers []*models.Follower
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

	if logErr := server.SyncLogData(); logErr != nil {
		log.Fatalf("Critical error reading log file: %v", err)
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
				fmt.Println("read error handle connection:", err)
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
			server.SendSnapshotToFollower(connection)
			follower := models.Follower{
				Conn: connection,
				Ch: make(chan string),
			}
			server.followers = append(server.followers, &follower)
			shouldCloseConnection = false
			server.followerMut.Unlock()

			server.FollowerListener(&follower)
			return;
		}

		commands := module.TextTokenizer(msg)

		shouldReturnToClient := true

		if commands[0] == "CRASH_TEST" {
			os.Exit(1) // Immediate exit, no defers run
		}

		if server.Role == enums.RoleLeader && commands[0] == "SET" {
			fmt.Println("Broadcasting SET command to followers")
			module.LogCommand(msg)
			shouldReturnToClient = len(server.followers) == 0 || server.BroadcastToFollowers(msg)
		}


		commandResult, err := module.CommandRouter(commands, server.inMemoryStore, server.Role)

		if err != nil {
			fmt.Println("Command error", err)
			connection.Write([]byte("ERR " + err.Error() + "\n"))
			// return
		} else if shouldReturnToClient {
			connection.Write([]byte(commandResult))
		}
	}

}

func (server *Server) FollowerListener(follower *models.Follower){
	defer func() {
        follower.Conn.Close()
        server.RemoveFollower(follower)
    }()

	reader := bufio.NewReader(follower.Conn)

	for {
		msg, err := reader.ReadString('\n')

		if err != nil {
			fmt.Println("Error Reading Follower Listener:", err)
			return
		}

		msg = strings.TrimSpace(msg)

		if(msg == "STORED"){
			select {
			case follower.Ch <- "STORED":
				fmt.Println("Reciving acknowledge command from follower")
			default:

			}
		}

	}
}

func (server *Server) RemoveFollower(f *models.Follower) {
    server.followerMut.Lock()
    defer server.followerMut.Unlock()

    for i, follower := range server.followers {
        // Compare pointers to find the exact follower
        if follower == f {
            // Standard Go way to remove an element from a slice
            server.followers = append(server.followers[:i], server.followers[i+1:]...)
            fmt.Println("Removed dead follower. Remaining:", len(server.followers))
            return
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

func(server *Server) BroadcastToFollowers(command string) bool{
	server.followerMut.Lock()
	followers := server.followers
	server.followerMut.Unlock()

	fmt.Println("Followers list", server.followers)

	var wg sync.WaitGroup
	askCount := make(chan bool, len(followers))
	
	
	for _, f := range server.followers {
		wg.Add(1)
		go func (follower *models.Follower)  {
			defer wg.Done()
			_, err := follower.Conn.Write([]byte(command + "\n"))

			if err != nil {
				fmt.Println("Failed to send command to follower ", err)
				return
			}


			select {
			case msg := <- follower.Ch:
				fmt.Println("Recieved message from channel", msg)
				if msg == "STORED"{
					askCount <- true
				}
			case <-time.After(2 * time.Second):
				fmt.Println("Broadcast follower error. Follower timed out")
			}
		}(f)
	}

	go func ()  {
		wg.Wait()
		close(askCount)
	}()

	for success := range askCount {
		fmt.Println("Total success", success)
		if success{
			fmt.Println("Total success returned", success)
			return true;
		}
	}

	return false
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

func(server *Server) SyncLogData() error{
	file, err := os.Open("wal.log")

	if err != nil {
		fmt.Println("Error opening wal.log for syncing: ", err)
		return err
	}

	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		commands := module.TextTokenizer(scanner.Text())

		fmt.Println("LOG COMMAND " + commands[0], commands[1])

		expiredDate, convertError := strconv.ParseInt(commands[4], 10, 64)

		if convertError != nil {
			return convertError
		}

		if time.Now().UnixNano() > expiredDate {
			fmt.Println(commands[1] + " is expired")
			continue
		}

		commandResult, errorCommand := module.CommandRouter(commands, server.inMemoryStore, server.Role)

		if errorCommand != nil {
			fmt.Println("Command Error Log ", errorCommand)
		}

		fmt.Println("Command Result ", commandResult)
	}

	return nil
}