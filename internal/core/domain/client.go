package core_domain

import (
	"time"

	core_transport "github.com/moond0wner/chat/internal/core/transport"
)

type Client struct {
	ID        int
	Name      string
	Conn      core_transport.Conn
	RoomID    int
	CreatedAt time.Time
}
