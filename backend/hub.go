package main

import (
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Room struct {
	ID          string
	Clients     map[*Client]bool
	VideoID     string
	VideoTitle  string
	Icebreakers []string
	LastState   PlayerState
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
			room, ok := h.rooms[client.roomId]
			h.mu.RUnlock()
			if ok {
				room.mu.Lock()
				room.Clients[client] = true
				room.mu.Unlock()
				log.Printf("Client %s registered to room %s", client.conn.RemoteAddr(), client.roomId)
			} else {
				log.Printf("Room %s not found for client %s", client.roomId, client.conn.RemoteAddr())
				client.conn.Close()
			}

		case client := <-h.unregister:
			h.mu.RLock()
			room, ok := h.rooms[client.roomId]
			h.mu.RUnlock()
			if ok {
				room.mu.Lock()
				if _, ok := room.Clients[client]; ok {
					delete(room.Clients, client)
					close(client.send)
					log.Printf("Client %s unregistered from room %s", client.conn.RemoteAddr(), client.roomId)
				}
				if len(room.Clients) == 0 {
					go h.scheduleRoomDeletion(room.ID)
				}
				room.mu.Unlock()
			}

		case message := <-h.broadcast:
			h.mu.RLock()
			room, ok := h.rooms[message.RoomID]
			h.mu.RUnlock()
			if ok {
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

func (h *Hub) createRoom(videoID, videoTitle string) (*Room, error) {
	roomID := uuid.New().String()[0:8]
	icebreakers, err := generateIcebreakers(videoTitle)
	if err != nil {
		log.Printf("AI generation failed: %v. Using default icebreakers.", err)
		icebreakers = []string{"What do you think will happen next?", "Who's your favorite character so far?", "Does this remind you of anything?"}
	}

	room := &Room{
		ID:          roomID,
		Clients:     make(map[*Client]bool),
		VideoID:     videoID,
		VideoTitle:  videoTitle,
		Icebreakers: icebreakers,
		LastState:   PlayerState{Status: -1, Time: 0},
	}

	h.mu.Lock()
	h.rooms[roomID] = room
	h.mu.Unlock()

	return room, nil
}

func (h *Hub) scheduleRoomDeletion(roomID string) {
	time.Sleep(5 * time.Minute)
	h.mu.Lock()
	defer h.mu.Unlock()

	room, ok := h.rooms[roomID]
	if !ok {
		return
	}

	room.mu.RLock()
	clientCount := len(room.Clients)
	room.mu.RUnlock()

	if clientCount == 0 {
		delete(h.rooms, roomID)
		log.Printf("Room %s deleted due to inactivity.", roomID)
	}
}
