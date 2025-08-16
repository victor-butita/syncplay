package main

import (
	"encoding/json"
	"log"
	"sync"
	"time"
)

// ... (Struct definitions are the same) ...
type Room struct {
	ID          string           `json:"id"`
	Clients     map[*Client]bool `json:"-"`
	VideoID     string           `json:"videoId"`
	VideoTitle  string           `json:"videoTitle"`
	Icebreakers []string         `json:"icebreakers"`
	LastState   PlayerState      `json:"lastState"`
	mu          sync.RWMutex
}
type Hub struct {
	rooms      map[string]*Room
	register   chan *Client
	unregister chan *Client
	broadcast  chan *Message
	mu         sync.RWMutex
}

func newHub() *Hub {
	return &Hub{
		rooms:      make(map[string]*Room),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan *Message),
	}
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.mu.RLock()
			room := h.rooms[client.roomId]
			h.mu.RUnlock()
			if room != nil {
				room.mu.Lock()
				room.Clients[client] = true
				room.mu.Unlock()
				// LOGGING: Client registration
				log.Printf("[HUB] Client registered to room %s. Total clients: %d", client.roomId, len(room.Clients))
			}

		case client := <-h.unregister:
			h.mu.RLock()
			room := h.rooms[client.roomId]
			h.mu.RUnlock()
			if room != nil {
				room.mu.Lock()
				if _, ok := room.Clients[client]; ok {
					delete(room.Clients, client)
					close(client.send)
					// LOGGING: Client unregistration
					log.Printf("[HUB] Client unregistered from room %s. Total clients: %d", client.roomId, len(room.Clients))
				}
				if len(room.Clients) == 0 {
					go h.scheduleRoomDeletion(room.ID)
				}
				room.mu.Unlock()
			}

		case message := <-h.broadcast:
			h.mu.RLock()
			room := h.rooms[message.RoomID]
			h.mu.RUnlock()
			if room != nil {
				room.mu.RLock()
				for client := range room.Clients {
					if client != message.Sender {
						select {
						case client.send <- message.Data:
						default:
							close(client.send)
							delete(room.Clients, client)
						}
					}
				}
				room.mu.RUnlock()
			}
		}
	}
}

func (h *Hub) getOrCreateRoom(roomId, videoId string) *Room {
	h.mu.Lock()
	defer h.mu.Unlock()

	if room, ok := h.rooms[roomId]; ok {
		// LOGGING: Room found
		log.Printf("[HUB] Found existing room %s", roomId)
		return room
	}

	room := &Room{
		ID:          roomId,
		Clients:     make(map[*Client]bool),
		VideoID:     videoId,
		VideoTitle:  "Loading title...",
		Icebreakers: []string{},
		LastState:   PlayerState{Status: -1, Time: 0},
	}
	h.rooms[roomId] = room
	// LOGGING: Room created
	log.Printf("[HUB] Created new room %s for video %s", roomId, videoId)

	go h.fetchRoomData(room)

	return room
}

func (h *Hub) fetchRoomData(room *Room) {
	// LOGGING: Starting async data fetch
	log.Printf("[ASYNC-%s] Starting data fetch for video %s", room.ID, room.VideoID)

	title, err := getYouTubeVideoInfo(room.VideoID)
	if err != nil {
		log.Printf("[ASYNC-%s] ERROR getting video info: %v", room.ID, err)
		title = "A YouTube Video"
	} else {
		log.Printf("[ASYNC-%s] SUCCESS got video title: \"%s\"", room.ID, title)
	}

	icebreakers, err := generateIcebreakers(title)
	if err != nil {
		log.Printf("[ASYNC-%s] ERROR generating AI icebreakers: %v", room.ID, err)
		icebreakers = []string{"What do you think of the video so far?"}
	} else {
		log.Printf("[ASYNC-%s] SUCCESS got AI icebreakers.", room.ID)
	}

	room.mu.Lock()
	room.VideoTitle = title
	room.Icebreakers = icebreakers
	room.mu.Unlock()

	payload, _ := json.Marshal(room)
	updateMessage, _ := json.Marshal(map[string]interface{}{
		"type":    "roomInfoUpdate",
		"payload": json.RawMessage(payload),
	})

	h.mu.RLock()
	roomFromMap := h.rooms[room.ID]
	h.mu.RUnlock()

	if roomFromMap != nil {
		roomFromMap.mu.RLock()
		clientCount := len(roomFromMap.Clients)
		for client := range roomFromMap.Clients {
			client.send <- updateMessage
		}
		roomFromMap.mu.RUnlock()
		// LOGGING: Sending update to clients
		log.Printf("[ASYNC-%s] Sent roomInfoUpdate to %d clients.", room.ID, clientCount)
	}
}

func (h *Hub) scheduleRoomDeletion(roomID string) {
	// LOGGING: Scheduling room for deletion
	log.Printf("[HUB] Room %s is empty. Scheduling for deletion in 5 minutes.", roomID)
	time.Sleep(5 * time.Minute)
	h.mu.Lock()
	defer h.mu.Unlock()
	if room, ok := h.rooms[roomID]; ok {
		if len(room.Clients) == 0 {
			delete(h.rooms, roomID)
			// LOGGING: Deleting room
			log.Printf("[HUB] DELETED room %s due to inactivity.", roomID)
		}
	}
}
