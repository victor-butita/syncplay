package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

// spaHandler implements the http.Handler interface and serves our Single Page Application.
type spaHandler struct {
	staticPath string
	indexPath  string
}

// ServeHTTP handles all requests. If a file exists at the path, it serves it.
// Otherwise, it serves the index.html file. This is the correct way to handle SPA routing.
func (h spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Get the absolute path to prevent directory traversal
	path, err := filepath.Abs(r.URL.Path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Prepend the static folder path
	path = filepath.Join(h.staticPath, path)

	// Check if the file exists
	_, err = os.Stat(path)
	if os.IsNotExist(err) || strings.HasSuffix(path, "/") {
		// File does not exist, serve index.html
		log.Printf("[HTTP-SPA] Serving index.html for path: %s", r.URL.Path)
		http.ServeFile(w, r, filepath.Join(h.staticPath, h.indexPath))
		return
	} else if err != nil {
		// If we got a different error, return a server error
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// LOGGING: Log static file serving
	log.Printf("[HTTP-STATIC] Serving static file: %s", r.URL.Path)
	// Otherwise, serve the static file
	http.FileServer(http.Dir(h.staticPath)).ServeHTTP(w, r)
}

func main() {
	// LOGGING: Announce server start
	log.Println("-----------------------------------------")
	log.Println("Initializing SyncPlay Server...")
	log.Println("-----------------------------------------")

	if err := godotenv.Load(); err != nil {
		log.Println("[WARNING] .env file not found. AI features require a GEMINI_API_KEY.")
	} else {
		log.Println("[INFO] .env file loaded successfully.")
	}

	hub := newHub()
	go hub.run()

	r := mux.NewRouter()

	// The WebSocket handler is now the only API endpoint
	r.HandleFunc("/ws/{roomId}", func(w http.ResponseWriter, r *http.Request) {
		serveWs(hub, w, r)
	})

	// This is the new, correct SPA handler
	spa := spaHandler{staticPath: "../frontend", indexPath: "index.html"}
	r.PathPrefix("/").Handler(spa)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("[INFO] Server starting on http://localhost:%s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("[FATAL] ListenAndServe error: %v", err)
	}
}
