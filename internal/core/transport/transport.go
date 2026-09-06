package core_transport

import (
	"context"
	"net"
	"time"
)

type Conn interface {
	Read([]byte) (int, error)
	Write([]byte) (int, error)
	Close() error
	SetWriteDeadline(time.Time) error
	LocalAddr() net.Addr
	RemoteAddr() net.Addr
}

type Server interface {
	Start(ctx context.Context) error
	Shutdown(ctx context.Context) error
}
