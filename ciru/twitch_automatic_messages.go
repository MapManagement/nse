package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

func newAutomaticMessages() *TwitchAutomaticMessages {
	automaticMessages := &TwitchAutomaticMessages{
		RWMutex:           &sync.RWMutex{},
		scheduledMessages: make(map[int]time.Time),
	}

	go automaticMessages.init()
	go automaticMessages.scheduler()

	return automaticMessages
}

func (automaticMessages *TwitchAutomaticMessages) init() {
	log.Info("Init Automatic messages")
	t := time.NewTicker(5 * time.Minute)
	for {
		select {
		case <-t.C:
			log.Info("Automatic messages ticker get messages from steve")
			automaticMessages.getMessages()
			automaticMessages.scheduleMessages()
		}
	}
}

func (automaticMessages *TwitchAutomaticMessages) getMessages() {
	r, err := http.Get(strings.Trim(os.Getenv("STEVE_URL"), " /") + "/automatic_message")
	if err != nil {
		log.Error("Automatic messages get messages from steve: ", err)
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Error("Automatic messages invalid body: ", err)
		return
	}

	automaticMessages.Lock()
	defer automaticMessages.Unlock()
	err = json.Unmarshal(body, &automaticMessages.messages)
	if err != nil {
		log.Error("Automatic messages unmarshal: ", err)
		return
	}
}

func (automaticMessages *TwitchAutomaticMessages) scheduler() {
	log.Info("Automatic messages started scheduler")
	t := time.NewTicker(30 * time.Second)
	for {
		select {
		case <-t.C:
			log.Info("Automatic messages looking for messages to send")
			automaticMessages.RLock()
			scheduledMessages := automaticMessages.scheduledMessages
			automaticMessages.RUnlock()
			for id, nextSend := range scheduledMessages {
				if nextSend.Before(time.Now()) {
					var message *TwitchAutomaticMessage
					for _, m := range automaticMessages.messages {
						if m.ID == id {
							message = m
							break
						}
					}

					if twitch.isOnline {
						log.Info("Automatic messages sending message ", message.ID)
						twitch.twirgo.SendMessage(twitch.twirgo.Options().DefaultChannel, message.Content)
					}
					automaticMessages.Lock()
					automaticMessages.scheduledMessages[id] = time.Now().Add(time.Duration(message.Interval) * time.Minute)
					automaticMessages.Unlock()
				}
			}
		}
	}
}

func (automaticMessages *TwitchAutomaticMessages) scheduleMessages() {
	log.Info("Automatic messages rescheduling all messages")
	automaticMessages.Lock()
	defer automaticMessages.Unlock()
	oldScheduler := automaticMessages.scheduledMessages
	automaticMessages.scheduledMessages = make(map[int]time.Time)

	for _, m := range automaticMessages.messages {
		if nextSend, ok := oldScheduler[m.ID]; ok {
			automaticMessages.scheduledMessages[m.ID] = nextSend
		} else {
			automaticMessages.scheduledMessages[m.ID] = time.Now().Add(time.Duration(m.Interval) * time.Minute)
		}
	}
}
