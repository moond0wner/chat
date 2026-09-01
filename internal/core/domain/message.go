package core_domain

import "time"

type Message struct {
	RoomID     int
	SenderName string
	SenderID   int
	Text       string
	IsSystem   bool
	SendAt     time.Time
}

type PrivateMessage struct {
	SenderID      int
	SenderName    string
	RecipentID    int
	RecipientName string
	Text          string
	SendAt        time.Time
}
