package main

import (
	"net/http"

	"github.com/gorilla/websocket"
)

type Hugo struct {
	hub *Hub
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

func newHugo() *Hugo {
	log.Info("Init Hugo")
	return &Hugo{
		hub: newHub(),
	}
}

func (hugo *Hugo) Serve(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error(err)
		return
	}

	client := &Client{
		conn: conn,
		send: make(chan []byte),
	}

	log.Info("New connection from client: ", conn.LocalAddr().String)
	hugo.hub.registerClient(client)

	go client.read()
	go client.write()
}
