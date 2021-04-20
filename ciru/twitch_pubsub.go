package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

func newTwitchPubSub() *TwitchPubSub {
	pb := &TwitchPubSub{
		closeConn:     make(chan bool),
		writeMessages: make(chan *TwitchPubSubRequest),
	}
	go pb.init()

	return pb
}

func (twitchPubSub *TwitchPubSub) init() {
	log.Info("Init PubSub")

	// reset to default
	// method could be called again due to reconnect
	twitchPubSub.readListenerClosed = false
	twitchPubSub.writeListenerClosed = false

	// we dont receive any information from pubsub if we can not authenticate
	twitch.RLock()
	accessToken := twitch.oauthToken.AccessToken
	twitch.RUnlock()

	if accessToken == "" {
		var stopWaiting bool

		// we are already in a goroutine, so we are not blocking anything
		t := time.NewTicker(30 * time.Second)
		for !stopWaiting {
			select {
			case <-t.C:
				twitch.RLock()
				if twitch.oauthToken != nil && twitch.oauthToken.AccessToken != "" {
					log.Info("PubSub: access token available, connecting to PubSub")
					stopWaiting = true
				} else {
					log.Info("PubSub: no access token available to authenticate, waiting for login")
				}
				twitch.RUnlock()
			}
		}
	}

	var err error
	twitchPubSub.conn, _, err = websocket.DefaultDialer.Dial("wss://pubsub-edge.twitch.tv", make(http.Header))
	if err != nil {
		log.Panic("Could not connect to Twitch PubSub: ", err)
	}
	log.Info("PubSub: connection established")

	// send ping messages to pubsub websocket every 4 1/2 minutes
	// should happen at least every 5 seconds
	cron.Stop("pubsub:_ping")
	cron.New("pubsub:_ping", twitchPubSub.ping, 270*time.Second)

	go twitchPubSub.readListener()
	go twitchPubSub.writeListener()
	go twitchPubSub.sendListenMessages()

	// reconnect if both listeners are closed
	go func(twitchPubSub *TwitchPubSub) {
		t := time.NewTicker(5 * time.Second)
		for {
			select {
			case <-t.C:
				if twitchPubSub.readListenerClosed && twitchPubSub.writeListenerClosed {
					go twitchPubSub.init()
					break
				}
			}
		}
	}(twitchPubSub)
}

func (twitchPubSub *TwitchPubSub) sendListenMessages() {
	log.Info("PubSub: send listen event")

	twitchPubSub.write(&TwitchPubSubRequest{
		Type: "LISTEN",
		Data: &TwitchPubSubRequestData{
			Topics:    []string{"channel-points-channel-v1." + twitch.channelID, "channel-subscribe-events-v1." + twitch.channelID, "channel-bits-events-v2." + twitch.channelID},
			AuthToken: twitch.oauthToken.AccessToken,
		},
	})
}

func (twitchPubSub *TwitchPubSub) ping() {
	log.Info("PubSub: sending PING")
	twitchPubSub.write(&TwitchPubSubRequest{
		Type: "PING",
	})
	twitchPubSub.lastPing = time.Now()
}

func (twitchPubSub *TwitchPubSub) readListener() {
	defer func(twitchPubSub *TwitchPubSub) {
		twitchPubSub.conn.Close()
		twitchPubSub.readListenerClosed = true
	}(twitchPubSub)

	for {
		_, message, err := twitchPubSub.conn.ReadMessage()
		if err != nil {
			log.Error("PubSub: read: ", err)
			return
		}

		log.Debug("PubSub: received: ", string(message))

		var r *TwitchPubSubResponse
		err = json.Unmarshal(message, &r)
		if err != nil {
			log.Error("PubSub: could not unmarshal response: ", err)
			continue
		}

		if r.Error != "" {
			log.Error("PubSub: Twitch error: ", err)
			continue
		}

		switch r.Type {
		case "RECONNECT":
			log.Info("PubSub: wants a reconnect")
			twitchPubSub.close()
			return

		case "PONG":
			log.Info("PubSub: received PONG")
			if twitchPubSub.lastPing.Add(10 * time.Second).Before(time.Now()) {
				twitchPubSub.close()
				return
			}

		case "MESSAGE":
			log.Info("PubSub: message received")
			r.Data.Message = strings.ReplaceAll(r.Data.Message, "\\\"", "\"")
			log.Debug("PubSub message: ", r.Data.Message)
			if strings.HasPrefix(r.Data.Topic, "channel-points-channel-v1") {
				var m TwitchPubSubMessageReward
				err := json.Unmarshal([]byte(r.Data.Message), &m)
				if err != nil {
					log.Error("PubSub: could not unmarshal message: ", err)
					continue
				}

				if strings.Contains(strings.ToLower(m.Data.Redemption.Reward.Title), "reputation") {
					twitchPubSub.addReputationPointsToUser(m.Data.Redemption.User.Login, m.Data.Redemption.Reward.Cost)
				}

				hugo.hub.broadcast(m)
			} else if strings.HasPrefix(r.Data.Topic, "channel-bits-events-v2") {
				var m TwitchPubSubMessageCheer
				err := json.Unmarshal([]byte(r.Data.Message), &m)
				if err != nil {
					log.Error("PubSub: could not unmarshal message: ", err)
					continue
				}

				twitchPubSub.addReputationPointsToUser(m.Data.Username, m.Data.BitsUsed*10)
			} else if strings.HasPrefix(r.Data.Topic, "channel-subscribe-events-v1") {
				log.Info("PubSub: new sub event")
				var m TwitchPubSubMessageSub
				err := json.Unmarshal([]byte(r.Data.Message), &m)
				if err != nil {
					log.Error("PubSub: could not unmarshal message: ", err)
					continue
				}

				if strings.HasPrefix(m.Context, "anon") {
					log.Debug("PubSub: sending 2500 reputation points to ", m.RecipientUserName)
					twitchPubSub.addReputationPointsToUser(m.RecipientUserName, 2500)
				} else if strings.HasSuffix(m.Context, "gift") {
					log.Debug("PubSub: sending 5000 reputation points to ", m.Username)
					twitchPubSub.addReputationPointsToUser(m.Username, 5000)
					log.Debug("PubSub: sending 2500 reputation points to ", m.RecipientUserName)
					twitchPubSub.addReputationPointsToUser(m.RecipientUserName, 2500)
				} else if strings.HasSuffix(m.Context, "sub") {
					log.Debug("PubSub: sending 2500 reputation points to ", m.RecipientUserName)
					twitchPubSub.addReputationPointsToUser(m.Username, 2500)
				}
			}
		}
	}
}

func (twitchPubSub *TwitchPubSub) writeListener() {
	defer func(twitchPubSub *TwitchPubSub) {
		twitchPubSub.conn.Close()
		twitchPubSub.writeListenerClosed = true
	}(twitchPubSub)

	for {
		select {
		case <-twitchPubSub.closeConn:
			log.Error("PubSub: closing connection")
			err := twitchPubSub.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("PubSub: write close:", err)
				return
			}
			return

		case message := <-twitchPubSub.writeMessages:
			log.Debugf("PubSub: sending: %+v", message)
			err := twitchPubSub.conn.WriteJSON(&message)
			if err != nil {
				log.Error("PubSub: write: ", err)
				return
			}
		}
	}
}

func (twitchPubSub *TwitchPubSub) write(r *TwitchPubSubRequest) {
	twitchPubSub.writeMessages <- r
}

func (twitchPubSub *TwitchPubSub) close() {
	twitchPubSub.closeConn <- true
}

func (twitchPubSub *TwitchPubSub) addReputationPointsToUser(username string, reputationPoints int) {
	requestURL := strings.Trim(os.Getenv("STEVE_URL"), " /") + "/user/" + username + "/reputation_points?reputation_points=" + strconv.Itoa(reputationPoints)
	req, err := http.NewRequest(http.MethodPut, requestURL, nil)
	log.Debug("Reputation points request url: ", requestURL)
	if err != nil {
		log.Error("Reputation points request prep: ", err)
		return
	}

	client := http.Client{}
	res, err := client.Do(req)
	if err != nil {
		log.Error("Reputation points request: ", err)
		return
	}

	if res.StatusCode != 200 {
		log.Error("Reputation points response: got ", res.StatusCode, " as response")
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			log.Error("Reputation points response read: ", err)
			return
		}
		log.Debugf("Reputation points response: %s", body)

		return
	}
}
