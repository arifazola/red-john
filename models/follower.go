package models

import (
	"net"
	"time"
)

type Follower struct {
	Conn net.Conn
	Ch chan string
	LastSeen time.Time
}