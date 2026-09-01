package core_domain

import (
	"net"
	"time"
)

type Client struct {
	ID        int
	Name      string
	Conn      net.Conn
	RoomID    int
	CreatedAt time.Time
}
