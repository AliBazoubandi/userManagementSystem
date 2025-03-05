package ws

import (
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

type Client struct {
	Conn     *websocket.Conn
	Message  chan *Message
	Logger   *zap.Logger
	ID       int32  `json:"id"`
	Username string `json:"username"`
	RoomID   int32  `json:"roomId"`
}

type Message struct {
	Content  string `json:"content"`
	Username string `json:"username"` // this is the username of the client who wants to send the message
	RoomID   int32  `json:"roomId"`
}

func (cl *Client) writeMessage() {
	defer func() {
		err := cl.Conn.Close()
		if err != nil {
			return
		}
	}()

	for {
		msg, ok := <-cl.Message
		if !ok {
			return
		}

		err := cl.Conn.WriteJSON(msg)
		if err != nil {
			return
		}
	}
}

func (cl *Client) readMessage(h *Hub) {
	defer func() {
		h.Unregister <- cl
		err := cl.Conn.Close()
		if err != nil {
			return
		}
	}()

	for {
		_, msg, err := cl.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				cl.Logger.Error("error: ", zap.Error(err))
			}
			break
		}

		newMsg := &Message{
			Content:  string(msg),
			Username: cl.Username,
			RoomID:   cl.RoomID,
		}

		h.BroadCast <- newMsg
	}
}
