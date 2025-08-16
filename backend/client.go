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
	// Allow all connections for this example. In production, you'd want a more restrictive policy.
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	hub    *Hub
	conn   *websocket.Conn
	send   chan []byte
	roomId string
}

// Message is a wrapper for data sent to the Hub's broadcast channel.
type Message struct {
	RoomID string
	Sender *Client
	Data   []byte
}

// PlayerState holds the synchronization data for the video player.
type PlayerState struct {
	Status int     `json:"status"`
	Time   float64 `json:"time"`
}

// serveWs handles websocket requests from the peer.
func serveWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	roomId := vars["roomId"]
	// The videoId is now passed as a query parameter for speed.
	videoId := r.URL.Query().Get("v")

	// LOGGING: New WebSocket connection attempt
	log.Printf("[WS] Connection attempt for room '%s' with video '%s'", roomId, videoId)

	if roomId == "" || videoId == "" {
		log.Printf("[WS-ERROR] Connection rejected: Missing room or video ID.")
		http.Error(w, "Missing room or video ID", http.StatusBadRequest)
		return
	}

	// This is the new core logic: get the room or create it if it's the first user.
	// This happens instantly.
	room := hub.getOrCreateRoom(roomId, videoId)

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[WS-ERROR] Failed to upgrade connection: %v", err)
		return
	}

	client := &Client{hub: hub, conn: conn, send: make(chan []byte, 256), roomId: roomId}
	client.hub.register <- client

	// Allow collection of memory referenced by the caller by doing all work in new goroutines.
	go client.writePump()
	go client.readPump()

	// Immediately send the current player state to the new client so they can sync up.
	room.mu.RLock()
	if room.LastState.Status != -1 { // -1 is the initial "unplayed" state.
		stateMsg, _ := json.Marshal(map[string]interface{}{
			"type":    "playerState",
			"payload": room.LastState,
		})
		client.send <- stateMsg
		// LOGGING: Sent initial state to new client
		log.Printf("[WS] Sent initial player state to new client in room %s", roomId)
	}
	room.mu.RUnlock()
}

// readPump pumps messages from the websocket connection to the hub.
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[WS-ERROR] Unexpected close error: %v", err)
			}
			break
		}

		// Before broadcasting, check if it's a state update and save it to the room.
		var msgData map[string]interface{}
		json.Unmarshal(message, &msgData)

		if msgType, ok := msgData["type"].(string); ok {
			// LOGGING: Received message from client
			log.Printf("[WS] Received '%s' message from client in room %s", msgType, c.roomId)
			if msgType == "playerState" {
				c.hub.mu.RLock()
				room := c.hub.rooms[c.roomId]
				c.hub.mu.RUnlock()

				if room != nil {
					var state PlayerState
					// We need to re-marshal and unmarshal the payload part of the message.
					if payloadBytes, err := json.Marshal(msgData["payload"]); err == nil {
						if json.Unmarshal(payloadBytes, &state) == nil {
							room.mu.Lock()
							room.LastState = state
							room.mu.Unlock()
						}
					}
				}
			}
		}

		// Broadcast the original message to other clients in the room.
		broadcastMessage := &Message{RoomID: c.roomId, Sender: c, Data: message}
		c.hub.broadcast <- broadcastMessage
	}
}

// writePump pumps messages from the hub to the websocket connection.
func (c *Client) writePump() {
	defer func() {
		c.conn.Close()
	}()
	for {
		message, ok := <-c.send
		if !ok {
			// The hub closed the channel.
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
