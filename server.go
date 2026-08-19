package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"math/rand/v2"
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
	ServerID string
	Addr, LeaderAddr, Role  string
	raftDataMut sync.Mutex
	RaftData *models.RaftData
	Term int
	timer *time.Timer
	followerMut sync.Mutex
	followers []*models.Follower
	nodesMut sync.Mutex
	nodes map[string]*models.Node
}

func(server *Server) StartServer(context context.Context) {
	fmt.Println("Start server as: ", server.Role)
	ln, err := net.Listen("tcp", ":"+server.Addr)

	server.nodes = make(map[string]*models.Node)

	node := models.Node{
		ID: server.ServerID,
		Address: server.Addr,
		Role: server.Role,
	}
	server.nodes[server.ServerID] = &node

	fmt.Printf("current nodes: %v\n", server.nodes[server.ServerID])

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
		log.Fatalf("Critical error reading log file: %v", logErr)
	}

	if server.Role == enums.RoleLeader {
		server.StartHeartbeatLoop() 
		server.FollowerCleaner()
	}

	
	for {
		conn, err := ln.Accept()

		if err != nil {
			fmt.Println("error connection ", err)
			return
		}

		fmt.Println("connected")

		go server.handleConnection(conn, context)
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

func(server *Server) handleConnection(connection net.Conn, context context.Context) {
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

		var messageModel models.Message

		err = json.Unmarshal([]byte(msg), &messageModel)

		// if err != nil {
		// 	fmt.Println("Error parsing message", err)
		// 	return
		// }


		if(err == nil){
			// connection.Write([]byte("YOU ARE SYNCED\n"))
			if(messageModel.Message == "VOTE_ME"){
				shouldCloseConnection = false
				fmt.Println("Follower asking for vote")
				var requestVoteRequest models.RequestVoteRequest
				err := json.Unmarshal([]byte(messageModel.Data), &requestVoteRequest)

				if err != nil {
					fmt.Println("Error parsing request vote request", err)
				}

				isVoteGranted := server.HandleElection(requestVoteRequest)

				requestVoteResponse := models.RequestVoteResponse{
					Term: requestVoteRequest.Term,
					VoteGranted: isVoteGranted,
				}

				stringifyResponse, err := json.Marshal(requestVoteResponse)
				if err != nil {
					fmt.Println("Error stringify requset vote response", err)
					return
				}

				_, err = connection.Write([]byte(string(stringifyResponse) + "\n"))

				if err != nil {
					fmt.Println("Cannot send vote granted response")
					return
				}

				// for {
				// 	newMsg, err := reader.ReadString('\n')

				// 	if err != nil {
				// 		fmt.Println("Error reading message vote")
				// 	}

				// 	fmt.Println("PING MESSAGE NEW", newMsg)
				// }

				// return
				continue
			} else if messageModel.Message == "NEW_LEADER"{
				server.ResetElectionTimer()
				server.Role = enums.RoleFollower
				server.ConnectToLeader(":"+messageModel.Data, context)
			} else {

				fmt.Println("Sending data to follower")
				server.nodesMut.Lock()
				var nodeModel models.Node
	
				err = json.Unmarshal([]byte(messageModel.Data), &nodeModel)
	
				server.nodes[nodeModel.ID] = &nodeModel
				server.nodesMut.Unlock()
	
				server.followerMut.Lock()
				follower := models.Follower{
					Conn: connection,
					Ch: make(chan string),
					LastSeen: time.Now(),
				}
	
				server.followers = append(server.followers, &follower)
				shouldCloseConnection = false
				server.followerMut.Unlock()
	
				server.SendSnapshotToFollower(connection)
				server.UpdateFollowersNodes()
				server.FollowerListener(&follower)
				return;
			}
		}

		commands := module.TextTokenizer(msg)

		shouldReturnToClient := true

		if commands[0] == "CRASH_TEST" {
			os.Exit(1) // Immediate exit, no defers run
		}

		if commands[0] == "PING"{
			server.HandleCommandFromLeader(msg, connection)
			continue
		}

		if server.Role == enums.RoleLeader && commands[0] == "SET" {
			fmt.Println("Broadcasting SET command to followers")
			module.LogCommand(msg)
			shouldReturnToClient = len(server.followers) == 0 || server.BroadcastToFollowers(msg)
		}


		fmt.Println("Command router as: ", server.Role)
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

		follower.LastSeen = time.Now()

		fmt.Println("Follower message: ", msg)

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

func (server *Server) FollowerCleaner(){
	ticker := time.NewTicker(10 * time.Second)

	go func ()  {
		for range ticker.C {
			server.followerMut.Lock()
			for _, f := range server.followers{
				if time.Since(f.LastSeen) > 60 * time.Second{
					fmt.Printf("Follower Cleaner: Follower %s is a zombie. Terminating.\n", f.Conn.RemoteAddr())

					f.Conn.Close()
				}
			}
			server.followerMut.Unlock()
		}
	}()
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
	data, err := server.serializeInMemoryData()

	if err != nil {
		fmt.Println("serialize error ", err)
		return err
	}

	inMemoryMessageModel := models.Message{
		Message: "SYNC_IN_MEMORY",
		Data: data,
	}

	stringifyInMemoryMessageModel, err := json.Marshal(inMemoryMessageModel)

	if err != nil {
		fmt.Println("Error parsing in memory message model", err)
		return err
	}

	conn.Write([]byte(string(stringifyInMemoryMessageModel) + "\n"))

	return nil
}

func(server *Server) serializeInMemoryData() (string, error){
	server.inMemoryStore.Mut.Lock()
	defer server.inMemoryStore.Mut.Unlock()

	data := server.inMemoryStore.GetAllUnsafe()

	json, err := json.Marshal(data)

	if err != nil {
		fmt.Println("error json marshal ", err)
		return "", err
	}

	return string(json), nil 
}

func(server *Server) serializeNodes() (string, error){
	server.nodesMut.Lock()
	defer server.nodesMut.Unlock()

	data := server.nodes

	json, err := json.Marshal(data)

	if err != nil {
		fmt.Println("error json marshal ", err)
		return "", err
	}

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

func(server *Server) UpdateFollowersNodes(){
	server.followerMut.Lock()
	followers := make([]*models.Follower, len(server.followers))
	copy(followers, server.followers)
	server.followerMut.Unlock()

	for _, item := range followers{
		go func (follower *models.Follower)  {
			nodeData, err := server.serializeNodes()

			if err != nil {
				fmt.Println("serialize error ", err)
			}

			nodeMessageModel := models.Message{
				Message: "NODES",
				Data: nodeData,
			}

			stringifyNodesMessageModel, err := json.Marshal(nodeMessageModel)

			if err != nil {
				fmt.Println("Error parsing in memory message model", err)
			}

			fmt.Println("Sending nodes data to follower", string(stringifyNodesMessageModel))

			follower.Conn.Write([]byte(string(stringifyNodesMessageModel) + "\n"))
		}(item)
	}
}

func (server *Server) StartHeartbeatLoop() {
	fmt.Println("Heartbeat loop started")
    ticker := time.NewTicker(3 * time.Second)
    go func() {
        for range ticker.C {
            server.followerMut.Lock()
            for _, f := range server.followers {
				fmt.Println("SENDING PING COMMAND", len(server.followers))
                _, err := f.Conn.Write([]byte("PING\n"))
                if err != nil {
                    fmt.Println("Ping failed for follower:", err)
                }
            }
            server.followerMut.Unlock()
        }
    }()
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

func(server *Server) ConnectToLeader(leaderAddr string, context context.Context) {
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

		nodeModel := models.Node{
			ID: server.ServerID,
			Address: server.Addr,
			Role: server.Role,
		}

		stringifyNode, err := json.Marshal(nodeModel)

		if err != nil {
			fmt.Println("Error parsing node model", err)
			return
		}

		messageModel := models.Message{
			Message: "SYNC_ME",
			Data: string(stringifyNode),
		}

		stringifyMessageModel, err := json.Marshal(messageModel)

		if err != nil {
			fmt.Println("Error parsing message model", err)
		}

		fmt.Println("Sending message to leader", string(stringifyMessageModel))

		_, writeError := conn.Write([]byte(string(stringifyMessageModel) + "\n"))

		if writeError != nil {
			fmt.Println("error writing message ", writeError)
			conn.Close()
			continue // continue to the next loop to retry 
		}

		go func ()  {
			<-context.Done()
			fmt.Println("Destroying follower connection")
			conn.Close()
		}()

		go func ()  {
			server.StartElectionTimer(context, conn)
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
			
			server.HandleCommandFromLeader(msg, conn)

			// fmt.Println("Command from leader", msg)

			// var event models.Event
			// jsonErr := json.Unmarshal([]byte(msg), &event)

			// //check if parsing the message from leader throws an error
			// //if message cannot be parsed, then it's a command (SET or GET)
			// //if message can be parsed, then it's a syncing data. Follower must store data from leader to it's own inMemoryStore
			// if jsonErr != nil {
			// 	commands := strings.Fields(msg)
				
			// 	if commands[0] == "PING"{
			// 		server.ResetElectionTimer()
			// 		conn.Write([]byte("PONG\n"))
			// 		continue
			// 	}

			// 	fmt.Println("Adding command for followers", commands)
			// 	result, _ := module.CommandRouter(commands, server.inMemoryStore, "")

			// 	fmt.Println("client command result ", result)
			// 	conn.Write([]byte("STORED\n"))
			// } else {

			// 	switch event.Message{
			// 	case "SYNC_IN_MEMORY":
			// 		var payloadString string
			// 		err = json.Unmarshal([]byte(string(event.Payload)), &payloadString)
			// 		if err != nil {
			// 			fmt.Println("Error parsing event payload nodes to string", err)
			// 		}

			// 		var parsedJson map[string]models.Item

			// 		_ = json.Unmarshal([]byte(payloadString), &parsedJson)
			// 		fmt.Println("Syncing")
			// 		server.inMemoryStore.SetAll(parsedJson)
			// 		fmt.Println("Finished syncing data")
			// 	case "NODES":
			// 		var payloadString string
			// 		err = json.Unmarshal([]byte(string(event.Payload)), &payloadString)
			// 		if err != nil {
			// 			fmt.Println("Error parsing event payload nodes to string", err)
			// 		}
					
			// 		var nodes map[string]*models.Node
			// 		fmt.Println("EVENT PAYLOAD", string(event.Payload))
			// 		err = json.Unmarshal([]byte(payloadString), &nodes)

			// 		if err != nil {
			// 			fmt.Println("Error parsing event payload nodes", err)
			// 		}
			// 		server.nodes = nodes
			// 	}
			// }
		}
	}

}

func(server *Server) HandleCommandFromLeader(msg string, conn net.Conn){
	fmt.Println("Command from leader", msg)

	var event models.Event
	jsonErr := json.Unmarshal([]byte(msg), &event)

	//check if parsing the message from leader throws an error
	//if message cannot be parsed, then it's a command (SET or GET)
	//if message can be parsed, then it's a syncing data. Follower must store data from leader to it's own inMemoryStore
	if jsonErr != nil {
		commands := strings.Fields(msg)
		
		if commands[0] == "PING"{
			server.ResetElectionTimer()
			conn.Write([]byte("PONG\n"))
			return
		}

		fmt.Println("Adding command for followers", commands)
		result, _ := module.CommandRouter(commands, server.inMemoryStore, "")

		fmt.Println("client command result ", result)
		conn.Write([]byte("STORED\n"))
	} else {

		switch event.Message{
		case "SYNC_IN_MEMORY":
			var payloadString string
			err := json.Unmarshal([]byte(string(event.Payload)), &payloadString)
			if err != nil {
				fmt.Println("Error parsing event payload nodes to string", err)
			}

			var parsedJson map[string]models.Item

			_ = json.Unmarshal([]byte(payloadString), &parsedJson)
			fmt.Println("Syncing")
			server.inMemoryStore.SetAll(parsedJson)
			fmt.Println("Finished syncing data")
		case "NODES":
			var payloadString string
			err := json.Unmarshal([]byte(string(event.Payload)), &payloadString)
			if err != nil {
				fmt.Println("Error parsing event payload nodes to string", err)
			}
			
			var nodes map[string]*models.Node
			fmt.Println("EVENT PAYLOAD", string(event.Payload))
			err = json.Unmarshal([]byte(payloadString), &nodes)

			if err != nil {
				fmt.Println("Error parsing event payload nodes", err)
			}
			server.nodes = nodes
		}
	}
}

func (server *Server) ResetElectionTimer(){
	fmt.Println(">>> RESET ELECTION TIMER")
	server.timer.Stop()
	min, max := 4000, 10000
	rangeNum := rand.IntN(max-min+1) + min
	server.timer.Reset(time.Duration(rangeNum) * time.Millisecond)
}

func (server *Server) StartElectionTimer(context context.Context, conn net.Conn){
	fmt.Println(">>> StartElectionTimer CALLED")
	min, max := 4, 10
	rangeNum := rand.IntN(max-min+1) + min
	server.RaftData.ElectionInterval = rangeNum 
	server.timer = time.NewTimer(time.Duration(server.RaftData.ElectionInterval) * time.Second)
	
	for {
		fmt.Println("Election timer start with interval:", server.RaftData.ElectionInterval)
		select{
		case <-context.Done():
			fmt.Println("Received cancellation context. Stopping election timer")
			return
		case <-server.timer.C:
			fmt.Println("Should start election")
			//Start election implementation

			fmt.Println("Follower term before send", server.RaftData.Term)
			server.Role = enums.RoleCandidate
			server.RaftData.Term++
			server.RaftData.VotedFor = server.ServerID
			server.RaftData.YesVote = 1
			server.RaftData.TotalVote = 1
			server.ResetElectionTimer()
			fmt.Println("Should send request votes")

			copiedNodes := make(map[string]*models.Node, len(server.nodes))

			server.nodesMut.Lock()
			copiedNodes = server.nodes
			server.nodesMut.Unlock()

			for _, item := range copiedNodes{
				if(item.Role == enums.RoleLeader){
					continue
				}
				
				if(item.ID == server.ServerID){
					continue
				}
				go func ()  {
					// address := server.nodes[id].Address
					
					fmt.Println("Sending Request Vote", item.Address)
					server.SendRequestVote(item.Address)

					if server.Role == enums.RoleLeader{
						fmt.Println("IM NEW LEADER")
						server.SendNewLeaderNotification(item.Address)
					}
				}()
			}

			// if server.Role == enums.RoleLeader{
			// 	copiedFollower := make([]*models.Follower, len(server.followers))
			// 	server.followerMut.Lock()
			// 	copiedFollower = server.followers
			// 	server.followerMut.Unlock()

			// 	for _, item := range copiedFollower{
			// 		go func ()  {
						
			// 		}
			// 	}
			// } else {
			// 	server.followers = nil
			// }

			return
		}
	}
}

func (server *Server) SendRequestVote(address string) {
	conn, err := net.Dial("tcp", ":"+address)

	if err != nil {
		fmt.Println("REQUEST VOTE ERROR: Cannot connect to node", err, address)
		return
	}

	defer conn.Close()

	// defer conn.Close()

	requestVote := models.RequestVoteRequest{
		Term: server.RaftData.Term,
		CandidateID: server.ServerID,
	}

	stringifyRequestVote, err := json.Marshal(requestVote)

	if err != nil {
		fmt.Println("Error stringify request vote", err)
		return
	}

	messageModel := models.Message {
		Message: "VOTE_ME",
		Data: string(stringifyRequestVote),
	}

	stringifyMessageModel, err := json.Marshal(messageModel)

	if err != nil {
		fmt.Println("Error stringify message model", err)
		return
	}

	_, writeError := conn.Write([]byte(string(stringifyMessageModel) + "\n"))
	// _, writeError := conn.Write([]byte("PING\n"))

	if writeError != nil {
		fmt.Println("error writing message ", writeError)
		conn.Close() 
	}

	majority := server.GetMajority()

	reader := bufio.NewReader(conn)

	for {
		msg, err := reader.ReadString('\n')

		if err != nil {
			fmt.Println("Error getting message")
			return
		}
		
		fmt.Println("MESSAGE FROM NEW LEADER", msg)

		if writeError != nil {
			fmt.Println("ERROR SENDING PING", err)
			return
		}

		var voteResponse models.RequestVoteResponse
		err = json.Unmarshal([]byte(msg), &voteResponse)

		if err != nil {
			fmt.Println("Error parsing vote response")
			return
		}

		fmt.Println("VOTE RESULT", voteResponse.VoteGranted)

		if voteResponse.VoteGranted {
			server.raftDataMut.Lock()
			server.RaftData.TotalVote ++
			server.RaftData.YesVote ++
			server.raftDataMut.Unlock()
		} else {
			server.RaftData.TotalVote ++
			server.RaftData.NoVote ++
		}

		fmt.Println("CURRENT VOTE", server.RaftData.TotalVote)
		fmt.Println("MAJORITY", majority)

		if server.RaftData.YesVote >= majority{
			server.Role = enums.RoleLeader
			server.StartHeartbeatLoop()
			fmt.Println("<><><><><> ELECTION COMPLETED", server.Role)
			// fmt.Println("SENDING PING AFTER VOTE")
			// _, writeError := conn.Write([]byte("FHDJSFHJDSFHJDSH\n"))
			// if writeError != nil {
			// 	fmt.Println("ERROR SENDING PING", err)
			// }
			// // server.FollowerListener(&follower)
			return
		}

		if server.RaftData.TotalVote >= majority || server.RaftData.TotalVote == len(server.nodes){
			fmt.Println("<><><><><> ELECTION COMPLETED", server.Role)
			return
		}
	}

}

func (server *Server) HandleElection(request models.RequestVoteRequest) bool{
	if(request.Term < server.RaftData.Term){
		fmt.Println("Vote rejected")
		return false
	}

	if(server.Role == enums.RoleLeader){
		fmt.Println("Leader cannot vote for election. REJECTED")
		return false
	}

	fmt.Println("server's term", server.RaftData.Term)
	fmt.Println("request's term", request.Term)

	if(request.Term > server.RaftData.Term){
		server.RaftData.Term = request.Term
		server.Role = enums.RoleFollower
		server.RaftData.VotedFor = ""

		fmt.Println("Update term, role, and voted for")
	}

	fmt.Println("This server has voted for", server.RaftData.VotedFor)

	if(server.RaftData.VotedFor == "" || server.RaftData.VotedFor == request.CandidateID){
		server.RaftData.VotedFor = request.CandidateID

		server.ResetElectionTimer()
		return true
	} else {
		fmt.Println("Vote rejected")
		return false
	}
}

func(server *Server) GetMajority() int{
	nodeSize := len(server.nodes)
	majority := math.Floor(float64(nodeSize) / 2) + 1
	return int(majority)
}

func (server *Server) SendNewLeaderNotification(address string){
	fmt.Println("Sending New Leader Notification To: ", address)
	conn, err := net.Dial("tcp", ":"+address)

	if err != nil {
		fmt.Println("REQUEST VOTE ERROR: Cannot connect to node", err, address)
		return
	}

	server.followerMut.Lock()
	follower := models.Follower{
		Conn: conn,
		Ch: make(chan string),
		LastSeen: time.Now(),
	}

	server.followers = append(server.followers, &follower)
	server.followerMut.Unlock()

	messageModel := models.Message {
		Message: "NEW_LEADER",
		Data: server.Addr,
	}

	stringifyMessageModel, err := json.Marshal(messageModel)

	if err != nil {
		fmt.Println("Error stringify new leader notificiation", err)
		return
	}

	conn.Write([]byte(string(stringifyMessageModel) + "\n"))

	conn.Close()
}