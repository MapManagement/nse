package main

import (
	"bytes"
	"encoding/json"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

type (
	Client struct {
		conn *websocket.Conn
		send chan []byte
	}

	ClientMessage struct {
		Content string `json:"content"`
	}
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

func (client *Client) read() {
	defer func() {
		hugo.hub.unregisterClient(client)
		client.conn.Close()
	}()

	client.conn.SetReadLimit(maxMessageSize)
	client.conn.SetReadDeadline(time.Now().Add(pongWait))
	client.conn.SetPongHandler(func(string) error { client.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })

	for {
		_, message, err := client.conn.ReadMessage()
		if err != nil {
			log.Info("Closing connection of: ", client.conn.LocalAddr().String)
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Errorf("%v", err)
			}
			break
		}

		message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))

		clientMessage := ClientMessage{}
		err = json.Unmarshal(message, &clientMessage)
		if err != nil {
			continue
		}

		log.Debug("Receiving message from client ", client.conn.LocalAddr().String, ": ", clientMessage.Content)

		twitch.twirgo.SendMessage(twitch.twirgo.Options().DefaultChannel, clientMessage.Content)
	}
}

func (client *Client) write() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		hugo.hub.unregisterClient(client)
		client.conn.Close()
	}()

	for {
		select {
		case message, ok := <-client.send:
			client.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel
				client.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := client.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			log.Debug("Sending message to client ", client.conn.LocalAddr().String, ": ", string(message))
			w.Write(message)

			// Add queued chat messages to the current websocket message.
			n := len(client.send)
			for i := 0; i < n; i++ {
				w.Write(newline)
				w.Write(<-client.send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			log.Debug("Ping client: ", client.conn.LocalAddr().String)
			client.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := client.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
