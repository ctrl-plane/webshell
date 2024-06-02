package server

import (
	"log"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/creack/pty"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Failed to upgrade connection:", err)
		return
	}
	defer conn.Close()

	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/bash"
	}

	cmd := exec.Command(shell)
	ptmx, err := pty.Start(cmd)
	if err != nil {
		log.Println("Failed to start pty:", err)
		return
	}
	defer func() { _ = ptmx.Close() }() // Best effort.

	// Set up a goroutine to read from the terminal and write to the WebSocket
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := ptmx.Read(buf)
			if err != nil {
				log.Println("Failed to read from pty:", err)
				break
			}
			if err := conn.WriteMessage(websocket.TextMessage, buf[:n]); err != nil {
				log.Println("Failed to write to WebSocket:", err)
				break
			}
		}
	}()

	// Set up a goroutine to read from the WebSocket and write to the terminal
	go func() {
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				log.Println("Failed to read from WebSocket:", err)
				break
			}
			if _, err := ptmx.Write(message); err != nil {
				log.Println("Failed to write to pty:", err)
				break
			}
		}
	}()

	// Keep the connection open
	for {
		time.Sleep(time.Second)
	}
}
func Serve() {
	http.HandleFunc("/ws", handleWebSocket)
	log.Println("Server started on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
