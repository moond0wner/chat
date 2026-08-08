package core_domain

import "net"

type Client struct {
	ID     string
	Name   string
	Conn   net.Conn
	RoomID string
}
