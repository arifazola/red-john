package models

import "net"

type Follower struct {
	Conn net.Conn
	Ch chan string
}