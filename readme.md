# Red John: Lightweight cache for Windows

---

Red John is a lightweight in-memory data storage built for Windows. It's purpose is similar to Redis, which is to store key value data in memory.

It's good if you are developing an app on Windows and need to store data in memory and don't want to be bothered installing Redis whether on Docker or VMs

# How To Run
1. Clone this project
2. Run `go build -o redjohn.exe` on root project directory
3. On terminal, run command `./redjohn.exe`. By default, it runs on port 8080. If you need to change the default port, you can go to main.go, change port number defined in this code `port := flag.String("port", "8080", "The port the server will listen to")`
4. To test, you can run command `telnet YOUR_IP_ADDRESS 8080` and type SET TEST_KEY TEST_VALUE. Or you can clone the Go client project [here](https://github.com/arifazola/red-john-go-client/blob/main/sample.go)

## Optional
Red John supports **leader-follower architecture**. To spawn followers server, simply run `./redjohn.exe --port=FOLLOWER_PORT --leader=LEADER_ADDRESS`

For example, if you have a leader that runs on server 93.199.74.18:8080, and you have another server with IP Address 42.30.110.38 that you want it to be follower, then on the follower's server, run command `./redjohn.exe --port=8002 --leader=93.199.74.18:8080`. As always, you can change the port.

