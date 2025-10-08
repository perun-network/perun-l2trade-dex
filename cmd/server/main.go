package main

import (
	"log"
	"net/http"

	"github.com/perun-network/perun-dex-websocket/internal/websocket"
)

func main() {
	// Create WebSocket hub
	hub := websocket.NewHub()
	go hub.Run()

	// Setup routes
	http.HandleFunc("/ws", hub.HandleWebSocket)
	http.HandleFunc("/", serveHome)

	// Serve static files
	fs := http.FileServer(http.Dir("./web/"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	log.Println("Perun DEX WebSocket server starting on :8080")
	log.Println("WebSocket endpoint: ws://localhost:8080/ws")
	log.Println("Web interface: http://localhost:8080")

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func serveHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	http.ServeFile(w, r, "web/index.html")
}
