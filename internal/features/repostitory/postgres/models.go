package history_postgres_repository

import (
	"time"
)

type MessageModel struct {
	RoomID     string
	SenderName string
	SenderID   string
	Text       string
	SendAt     time.Time
}
