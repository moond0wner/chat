package core_domain

type Room struct {
	Name    string
	Clients map[string]*Client
}

func NewRoom(name string) *Room {
	return &Room{
		Name:    name,
		Clients: make(map[string]*Client),
	}
}
