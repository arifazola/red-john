package main

import (
	"bufio"
	"fmt"
	"net"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// func TestLeaderElection(t *testing.T) {
//     leader := startServer(t, "8081", "")
//     defer leader.Process.Kill()

//     follower1 := startServer(t, "8082", "localhost:8081")
//     defer follower1.Process.Kill()

//     follower2 := startServer(t, "8083", "localhost:8081")
//     defer follower2.Process.Kill()

//     time.Sleep(2 * time.Second)

//     // Test your cluster here.
// }

func TestSetCommandOneServerActive_Success(t *testing.T){
	startServer(t, "8080", "")

	conn, err := net.Dial("tcp", "localhost:8080")

	if err != nil {
		t.Fatal("Error connect to leader", err)
	}

	defer conn.Close()

	_, err = conn.Write([]byte("SET NAME ARI\n"))

	if err != nil {
		t.Fatal("Error Making Set Command", err)
	}

	reader := bufio.NewReader(conn)

	response, err := reader.ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}

	responseMsg := strings.TrimSpace(response)

	assert.Equal(t, "OK", responseMsg, "Message should be OK")

}

func TestSetCommandMultipleServerActive_BroadcastSuccess(t *testing.T){
	startServer(t, "8080", "")

	startServer(t, "8081", "localhost:8080")

    startServer(t, "8082", "localhost:8080")

    startServer(t, "8082", "localhost:8080")
    

	conn, err := net.Dial("tcp", "localhost:8080")

	if err != nil {
		t.Fatal("Error connect to leader", err)
	}

	defer conn.Close()

	_, err = conn.Write([]byte("SET NAME ARI\n"))

	if err != nil {
		t.Fatal("Error Making Set Command", err)
	}

	reader := bufio.NewReader(conn)

	response, err := reader.ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}

	responseMsg := strings.TrimSpace(response)

	assert.Equal(t, "OK", responseMsg, "Message should be OK")

}

func TestSetCommandMultipleServerActive_ShuouldReturnOKWhenOneFollowerIsDeadDuringBroadcast(t *testing.T){
	startServer(t, "8080", "")

	startServer(t, "8081", "localhost:8080")

    server3 := startServer(t, "8082", "localhost:8080")

    startServer(t, "8083", "localhost:8080")
    
	conn, err := net.Dial("tcp", "localhost:8080")

	if err != nil {
		t.Fatal("Error connect to leader", err)
	}

	defer conn.Close()

	time.Sleep(5 * time.Second)

	go func ()  {
		fmt.Println("KILLING PROCESS")
		server3.Process.Kill()
	}()

	channel := make(chan string)
	go func ()  {
		_, err = conn.Write([]byte("SET NAME ARI\n"))

		if err != nil {
			fmt.Println("Error Making Set Command", err)
			channel <- ""
			return
		}

		reader := bufio.NewReader(conn)

		response, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading line", err)
			channel <- ""
			return
		}

		responseMsg := strings.TrimSpace(response)

		channel <- responseMsg
	}()
	

	assert.Equal(t, "OK", <-channel, "Message should be OK")

}

func TestSetCommandOneServerActive_InvalidCommand(t *testing.T){
	startServer(t, "8080", "")

	conn, err := net.Dial("tcp", "localhost:8080")

	if err != nil {
		t.Fatal("Error connect to leader", err)
	}

	defer conn.Close()

	_, err = conn.Write([]byte("SET NAME\n"))

	if err != nil {
		t.Fatal("Error Making Set Command", err)
	}

	reader := bufio.NewReader(conn)

	response, err := reader.ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}

	responseMsg := strings.TrimSpace(response)

	assert.NotEqual(t, "OK", responseMsg, "Message should not be OK")

}

func TestGetCommandOneServerActive_Success(t *testing.T){
	startServer(t, "8080", "")

	conn, err := net.Dial("tcp", "localhost:8080")

	if err != nil {
		t.Fatal("Error connect to leader", err)
	}

	_, err = conn.Write([]byte("SET NAME ARI\n"))

	if err != nil {
		t.Fatal("Error Making Set Command", err)
	}

	conn.Close()

	conn, err = net.Dial("tcp", "localhost:8080")

	_, err = conn.Write([]byte("GET NAME\n"))

	reader := bufio.NewReader(conn)

	response, err := reader.ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}

	responseMsg := strings.TrimSpace(response)

	assert.Equal(t, "ARI", responseMsg, "Message should be ARI")

	conn.Close()
}

func TestGetCommandOneServerActive_NoData(t *testing.T){
	startServer(t, "8080", "")

	conn, err := net.Dial("tcp", "localhost:8080")

	if err != nil {
		t.Fatal("Error connect to leader", err)
	}

	_, err = conn.Write([]byte("SET NAME ARI\n"))

	if err != nil {
		t.Fatal("Error Making Set Command", err)
	}

	conn.Close()

	conn, err = net.Dial("tcp", "localhost:8080")

	_, err = conn.Write([]byte("GET city\n"))

	reader := bufio.NewReader(conn)

	response, err := reader.ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}

	responseMsg := strings.TrimSpace(response)

	fmt.Println("RESPONSE", responseMsg)

	assert.Equal(t, "NOT_FOUND", responseMsg, "Message should be NOT_FOUND")

	conn.Close()
}

func TestConcurentSetCommandOneServerActive_Success(t *testing.T){
	startServer(t, "8080", "")

	channel := make(chan int)
	var wg sync.WaitGroup
	wg.Add(2)
	go func ()  {
		
		conn, err := net.Dial("tcp", "localhost:8080")

		if err != nil {
			channel <- 0
		}

		
		defer func ()  {
			wg.Done()
			conn.Close()
		}()

		_, err = conn.Write([]byte("SET NAME ARI\n"))

		if err != nil {
			channel <- 0
		}

		reader := bufio.NewReader(conn)

		response, err := reader.ReadString('\n')
		if err != nil {
			channel <- 0
		}

		responseMsg := strings.TrimSpace(response)

		if responseMsg == "OK"{
			channel <- 1
		} else {
			channel <- 0
		}
	}()

	go func ()  {
		conn, err := net.Dial("tcp", "localhost:8080")

		if err != nil {
			channel <- 0
		}

		
		defer func ()  {
			wg.Done()
			conn.Close()
		}()

		_, err = conn.Write([]byte("SET COUNTRY IDN\n"))

		if err != nil {
			channel <- 0
		}

		reader := bufio.NewReader(conn)

		response, err := reader.ReadString('\n')
		if err != nil {
			channel <- 0
		}

		responseMsg := strings.TrimSpace(response)

		if responseMsg == "OK"{
			channel <- 1
		} else {
			channel <- 0
		}
	}()

	go func ()  {
		wg.Wait()
		close(channel)
	}()

	
	allSetPassed := true
	for value := range channel{
		if value == 0 {
			allSetPassed = false
			break
		}

		fmt.Println("CHANNEL VALUE", value)
	}

	fmt.Println("ALL CHANNEL HAVE BEEN RECEIVED")


	assert.Equal(t, true, allSetPassed, "All Set Command Should Be OK")

}

func TestConcurentSetCommandOneServerActive_OneFail(t *testing.T){
	startServer(t, "8080", "")

	channel := make(chan int)
	var wg sync.WaitGroup
	wg.Add(2)
	go func ()  {
		
		conn, err := net.Dial("tcp", "localhost:8080")

		if err != nil {
			channel <- 0
		}

		
		defer func ()  {
			wg.Done()
			conn.Close()
		}()

		_, err = conn.Write([]byte("SET NAME ARI\n"))

		if err != nil {
			channel <- 0
		}

		reader := bufio.NewReader(conn)

		response, err := reader.ReadString('\n')
		if err != nil {
			channel <- 0
		}

		responseMsg := strings.TrimSpace(response)

		if responseMsg == "OK"{
			channel <- 1
		} else {
			channel <- 0
		}
	}()

	go func ()  {
		conn, err := net.Dial("tcp", "localhost:8080")

		if err != nil {
			channel <- 0
		}

		
		defer func ()  {
			wg.Done()
			conn.Close()
		}()

		_, err = conn.Write([]byte("ST COUNTRY IDN\n"))

		if err != nil {
			channel <- 0
		}

		reader := bufio.NewReader(conn)

		response, err := reader.ReadString('\n')
		if err != nil {
			channel <- 0
		}

		responseMsg := strings.TrimSpace(response)

		if responseMsg == "OK"{
			channel <- 1
		} else {
			channel <- 0
		}
	}()

	go func ()  {
		wg.Wait()
		close(channel)
	}()

	
	allSetPassed := true
	for value := range channel{
		if value == 0 {
			allSetPassed = false
			break
		}

		fmt.Println("CHANNEL VALUE", value)
	}

	fmt.Println("ALL CHANNEL HAVE BEEN RECEIVED")


	assert.Equal(t, false, allSetPassed, "Return should be false because one command is fail")

}

func TestConcurentGetCommandOneServerActive_SuccessAll(t *testing.T){
	startServer(t, "8080", "")

	channel := make(chan int)
	var wg sync.WaitGroup
	wg.Add(2)
	go func ()  {
		defer wg.Done()
		conn, err := net.Dial("tcp", "localhost:8080")

		if err != nil {
			channel <- 0
			return
		}

		_, err = conn.Write([]byte("SET NAME ARI\n"))

		if err != nil {
			channel <- 0
			return
		}

		conn.Close()

		conn, err = net.Dial("tcp", "localhost:8080")

		_, err = conn.Write([]byte("GET NAME\n"))

		reader := bufio.NewReader(conn)

		response, err := reader.ReadString('\n')
		if err != nil {
			channel <- 0
			return
		}

		responseMsg := strings.TrimSpace(response)

		if responseMsg == "ARI"{
			channel <- 1
		} else {
			channel <- 0
		}

		conn.Close()
	}()

	go func ()  {
		defer wg.Done()
		conn, err := net.Dial("tcp", "localhost:8080")

		if err != nil {
			channel <- 0
			return
		}

		_, err = conn.Write([]byte("SET COUNTRY IDN\n"))

		if err != nil {
			channel <- 0
			return
		}

		conn.Close()

		conn, err = net.Dial("tcp", "localhost:8080")

		_, err = conn.Write([]byte("GET COUNTRY\n"))

		reader := bufio.NewReader(conn)

		response, err := reader.ReadString('\n')
		if err != nil {
			channel <- 0
			return
		}

		responseMsg := strings.TrimSpace(response)

		if responseMsg == "IDN"{
			channel <- 1
		} else {
			channel <- 0
		}

		conn.Close()
	}()

	go func ()  {
		wg.Wait()
		close(channel)
	}()

	
	allGetPassed := true
	for value := range channel{
		if value == 0 {
			allGetPassed = false
			break
		}

		fmt.Println("CHANNEL VALUE", value)
	}

	fmt.Println("ALL CHANNEL HAVE BEEN RECEIVED")


	assert.Equal(t, true, allGetPassed, "Return should be true because all get command return data")

}

func TestConcurentGetCommandOneServerActive_PartialSuccess(t *testing.T){
	startServer(t, "8080", "")

	channel := make(chan int)
	var wg sync.WaitGroup
	wg.Add(2)
	go func ()  {
		defer wg.Done()
		conn, err := net.Dial("tcp", "localhost:8080")

		if err != nil {
			channel <- 0
			return
		}

		_, err = conn.Write([]byte("SET NAME ARI\n"))

		if err != nil {
			channel <- 0
			return
		}

		conn.Close()

		conn, err = net.Dial("tcp", "localhost:8080")

		_, err = conn.Write([]byte("GET NAME\n"))

		reader := bufio.NewReader(conn)

		response, err := reader.ReadString('\n')
		if err != nil {
			channel <- 0
			return
		}

		responseMsg := strings.TrimSpace(response)

		if responseMsg == "ARI"{
			channel <- 1
		} else {
			channel <- 0
		}

		conn.Close()
	}()

	go func ()  {
		defer wg.Done()
		conn, err := net.Dial("tcp", "localhost:8080")

		if err != nil {
			channel <- 0
			return
		}

		_, err = conn.Write([]byte("SET COUNTRY IDN\n"))

		if err != nil {
			channel <- 0
			return
		}

		conn.Close()

		conn, err = net.Dial("tcp", "localhost:8080")

		_, err = conn.Write([]byte("GET STREET\n"))

		reader := bufio.NewReader(conn)

		response, err := reader.ReadString('\n')
		if err != nil {
			channel <- 0
			return
		}

		responseMsg := strings.TrimSpace(response)

		if responseMsg == "NOT_FOUND"{
			channel <- 1
		} else {
			channel <- 0
		}

		conn.Close()
	}()

	go func ()  {
		wg.Wait()
		close(channel)
	}()

	
	allGetPassed := true
	for value := range channel{
		if value == 0 {
			allGetPassed = false
			break
		}

		fmt.Println("CHANNEL VALUE", value)
	}

	fmt.Println("ALL CHANNEL HAVE BEEN RECEIVED")


	assert.Equal(t, true, allGetPassed, "Return should be true because all one get command return correct data and one return not found")

}

func startServer(t *testing.T, port string, leader string) *exec.Cmd {
	args := []string{
		"--port", port,
	}

	if leader != "" {
		args = append(args, "--leader", leader)
	}

	cmd := exec.Command("./redjohn.exe", args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}

	cmd.Stderr = cmd.Stdout

	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}

	// Print server output while test is running.
	go func() {
		scanner := bufio.NewScanner(stdout)

		for scanner.Scan() {
			fmt.Printf("[server:%s] %s\n", port, scanner.Text())
		}
	}()

	return cmd
}