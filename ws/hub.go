package ws

import "sync"

type Hub struct {
	mu         sync.Mutex
	Rooms      map[int32]*Room
	Register   chan *Client
	Unregister chan *Client
	BroadCast  chan *Message
}

type Room struct {
	ID      int32             `json:"id"`
	Name    string            `json:"name"`
	Clients map[int32]*Client `json:"clients"`
}

func NewHub() *Hub {
	return &Hub{
		Rooms:      make(map[int32]*Room),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		BroadCast:  make(chan *Message, 5),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case cl := <-h.Register:
			if _, ok := h.Rooms[cl.RoomID]; ok {
				if _, ok := h.Rooms[cl.RoomID].Clients[cl.ID]; !ok {
					h.Rooms[cl.RoomID].Clients[cl.ID] = cl
				}
			}

		case cl := <-h.Unregister:
			if _, ok := h.Rooms[cl.RoomID]; ok {
				if _, ok := h.Rooms[cl.RoomID].Clients[cl.ID]; ok {
					if len(h.Rooms[cl.RoomID].Clients) != 0 {
						h.BroadCast <- &Message{
							Content:  "User Left The Room",
							RoomID:   cl.RoomID,
							Username: cl.Username,
						}
					}

					delete(h.Rooms[cl.RoomID].Clients, cl.ID)
					close(cl.Message)
				}
			}
		case msg := <-h.BroadCast:
			if _, ok := h.Rooms[msg.RoomID]; ok {
				for _, cl := range h.Rooms[msg.RoomID].Clients {
					cl.Message <- msg
				}
			}
		}
	}
}
