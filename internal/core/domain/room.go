package core_domain

type Room struct {
	ID      int
	Name    string
	Clients map[int]*Client
}

func NewRoom(name string) *Room {
	return &Room{
		Name:    name,
		Clients: make(map[int]*Client),
	}
}
