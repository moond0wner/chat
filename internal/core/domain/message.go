package core_domain

type Message struct {
	RoomID     string
	SenderName string
	SenderID   string
	Text       string
	// sendAt time.Time - на будущее для хранения в БД
}

type PrivateMessage struct {
	SenderID      string
	SenderName    string
	RecipentID    string
	RecipientName string
	Text          string
	// sendAt time.Time - на будущее для хранения в БД
}
