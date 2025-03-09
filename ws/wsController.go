package ws

import (
	"main/db"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

type WsController struct {
	Queries *db.Queries
	hub     *Hub
	logger  *zap.Logger
}

func NewWsController(queries *db.Queries, h *Hub, l *zap.Logger) *WsController {
	return &WsController{
		Queries: queries,
		hub:     h,
		logger:  l,
	}
}

func (ws *WsController) CreateRoom(c *gin.Context) {

	username, exists := c.Get("username")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	type RequestBody struct {
		Name string `json:"name" binding:"required"`
	}

	// Bind the request body to a map first
	var raw map[string]interface{}
	if err := c.ShouldBindJSON(&raw); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Ensure that the only field present is "name"
	if len(raw) > 1 || (len(raw) == 1 && raw["name"] == nil) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Request body contains unexpected fields"})
		return
	}

	// Now bind the valid data to your struct
	var req RequestBody
	name, ok := raw["name"].(string)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Request body contains an invalid field"})
		return
	}
	req.Name = name

	room, err := ws.Queries.CreateRoom(c.Request.Context(), req.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	addUser := db.AddUserToRoomParams{
		Username: username.(string),
		RoomID:   pgtype.Int4{Int32: room.ID, Valid: true},
	}

	_, err = ws.Queries.AddUserToRoom(c.Request.Context(), addUser)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cant assign the user to the room"})
		return
	}

	ws.hub.mu.Lock()
	ws.hub.Rooms[room.ID] = &Room{
		ID:      room.ID,
		Name:    room.Name,
		Clients: make(map[int32]*Client),
	}
	ws.hub.mu.Unlock()

	c.JSON(http.StatusCreated, room)
}

var Upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (ws *WsController) JoinRoom(c *gin.Context) {
	usernameRaw, exists := c.Get("username")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	username, ok := usernameRaw.(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error parsing username from header"})
		return
	}

	conn, err := Upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	roomID := c.Param("roomId")
	if roomID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Room ID is required"})
		return
	}
	roomIdInt, err := strconv.Atoi(roomID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid room ID"})
		return
	}

	user, err := ws.Queries.GetUserByUsername(c.Request.Context(), username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not retrieve user information"})
		return
	}

	cl := &Client{
		Conn:     conn,
		Message:  make(chan *Message, 10),
		Logger:   ws.logger,
		ID:       user.ID,
		Username: username,
		RoomID:   int32(roomIdInt),
	}

	msg := &Message{
		Content:  "A New User Joined The Room",
		RoomID:   int32(roomIdInt),
		Username: username,
	}

	ws.hub.Register <- cl
	go cl.writeMessage()
	go cl.readMessage(ws.hub)
	ws.hub.BroadCast <- msg

}

func (ws *WsController) GetRooms(c *gin.Context) {
	rooms, err := ws.Queries.GetRooms(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, rooms)
}

type ClientRes struct {
	ID       int32  `json:"id"`
	Username string `json:"username"`
}

func (ws *WsController) GetClients(c *gin.Context) {
	roomId := c.Param("roomId")
	if roomId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Room ID is required"})
		return
	}
	roomIdInt, err := strconv.Atoi(roomId)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid room ID"})
		return
	}
	_, err = ws.Queries.GetRoomById(c.Request.Context(), int32(roomIdInt))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Room not found"})
		return
	}

	users, err := ws.Queries.GetUsersByRoomID(c.Request.Context(), pgtype.Int4{Int32: int32(roomIdInt), Valid: true})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	clients := make([]ClientRes, 0)

	for _, client := range users {
		clients = append(clients, ClientRes{
			ID:       client.ID,
			Username: client.Username,
		})
	}

	c.JSON(http.StatusOK, clients)
}
