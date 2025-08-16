package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

type Client struct {
	hub    *Hub
	conn   *websocket.Conn
	send   chan []byte
	roomId string
}

type Message struct {
	RoomID string
	Sender *Client
	Data   []byte
}

type PlayerState struct {
	Status int     `json:"status"`
	Time   float64 `json:"time"`
}

func serveWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	roomId := vars["roomId"]

	// CORRECTED: Called methods on the 'mu' field of the hub
	hub.mu.RLock()
	room, ok := hub.rooms[roomId]
	hub.mu.RUnlock()

	if !ok {
		http.NotFound(w, r)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	client := &Client{hub: hub, conn: conn, send: make(chan []byte, 256), roomId: roomId}
	client.hub.register <- client

	go client.writePump()
	go client.readPump()

	initialState, _ := json.Marshal(map[string]interface{}{
		"type":        "initialState",
		"videoID":     room.VideoID,
		"videoTitle":  room.VideoTitle,
		"icebreakers": room.Icebreakers,
		"playerState": room.LastState,
	})
	client.send <- initialState
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			break
		}

		var msgData map[string]interface{}
		json.Unmarshal(message, &msgData)

		if msgData["type"] == "playerState" {
			// CORRECTED: Called methods on the 'mu' field of the hub
			c.hub.mu.RLock()
			room := c.hub.rooms[c.roomId]
			c.hub.mu.RUnlock()

			if room != nil {
				var state PlayerState
				stateData, _ := json.Marshal(msgData["payload"])
				json.Unmarshal(stateData, &state)
				room.mu.Lock()
				room.LastState = state
				room.mu.Unlock()
			}
		}

		broadcastMessage := &Message{RoomID: c.roomId, Sender: c, Data: message}
		c.hub.broadcast <- broadcastMessage
	}
}

func (c *Client) writePump() {
	defer func() {
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)
			if err := w.Close(); err != nil {
				return
			}
		}
	}
}
