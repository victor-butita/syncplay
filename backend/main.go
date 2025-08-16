package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

type CreateRoomRequest struct {
	URL string `json:"url"`
}

func createRoomHandler(hub *Hub, w http.ResponseWriter, r *http.Request) {
	var req CreateRoomRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "Invalid request body"}`, http.StatusBadRequest)
		return
	}

	videoID := getYouTubeID(req.URL)
	if videoID == "" {
		http.Error(w, `{"error": "Invalid or unsupported YouTube URL"}`, http.StatusBadRequest)
		return
	}

	videoTitle, err := getYouTubeVideoInfo(videoID)
	if err != nil {
		log.Printf("Failed to get video info for ID %s: %v", videoID, err)
		http.Error(w, `{"error": "Could not retrieve video information"}`, http.StatusInternalServerError)
		return
	}

	room, err := hub.createRoom(videoID, videoTitle)
	if err != nil {
		log.Printf("Failed to create room with AI: %v", err)
		http.Error(w, `{"error": "Failed to generate room details"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"roomId": room.ID})
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("Note: .env file not found, expecting environment variables.")
	}

	hub := newHub()
	go hub.run()

	r := mux.NewRouter()

	r.HandleFunc("/create", func(w http.ResponseWriter, r *http.Request) {
		createRoomHandler(hub, w, r)
	}).Methods("POST")

	r.HandleFunc("/ws/{roomId}", func(w http.ResponseWriter, r *http.Request) {
		serveWs(hub, w, r)
	})

	staticFileServer := http.FileServer(http.Dir("../frontend"))
	r.PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		staticPath := "../frontend" + r.URL.Path
		if _, err := os.Stat(staticPath); os.IsNotExist(err) {
			http.ServeFile(w, r, "../frontend/index.html")
		} else {
			staticFileServer.ServeHTTP(w, r)
		}
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Println("Server starting on http://localhost:" + port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("ListenAndServe error: %v", err)
	}
}
