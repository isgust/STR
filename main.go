package main

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for local game
	},
}

func main() {
	game := NewGame()
	go game.Run()

	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/", fs)

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("upgrade err:", err)
			return
		}
		
		client := &Client{
			conn: conn,
			game: game,
			send: make(chan []byte, 256),
		}
		
		game.register <- client
		go client.readPump()
		go client.writePump()
	})

	log.Println("Servidor rodando na porta :8080...")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
