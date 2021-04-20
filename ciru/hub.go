package main

import (
	"encoding/json"
	"sync"
)

type (
	Hub struct {
		*sync.RWMutex
		clients map[*Client]bool
	}

	Data struct {
		Type string      `json:"type"`
		Data interface{} `json:"data"`
	}
)

func newHub() *Hub {
	log.Info("Init Hub")
	return &Hub{
		clients: make(map[*Client]bool),
		RWMutex: &sync.RWMutex{},
	}
}

func (hub *Hub) registerClient(client *Client) {
	log.Debug("Register new client")
	hub.Lock()
	hub.clients[client] = true
	hub.Unlock()
}

func (hub *Hub) unregisterClient(client *Client) {
	hub.Lock()
	defer hub.Unlock()
	if _, ok := hub.clients[client]; ok {
		log.Debug("Unregister new client")
		delete(hub.clients, client)
		close(client.send)
	}
}

func (hub *Hub) broadcast(data interface{}) {
	var t string
	switch d := data.(type) {
	case TwitchMessage:
		t = "message"
	case TwitchClearchat:
		t = "clearchat"
	case TwitchClearmsg:
		t = "clearmsg"
	case TwitchPubSubMessageReward:
		t = d.Type
	case TwitchPubSubMessageSub:
		t = "sub"
	default:
		log.Error("Got invalid type to broadcast")
		return
	}

	d := Data{
		Type: t,
		Data: data,
	}

	json, err := json.Marshal(d)
	if err != nil {
		log.Error(err)
		return
	}

	log.Debug("Broadcast message: ", string(json))
	for client := range hub.clients {
		client.send <- []byte(json)
		log.Debug("Queued data for: ", client.conn.UnderlyingConn().RemoteAddr().String)
	}
}
