package main

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/curiTTV/twirgo"
)

func (twitch *Twitch) eventMessageReceived(t *twirgo.Twitch, event twirgo.EventMessageReceived) {
	m := TwitchMessage{
		ID:          event.Message.ID,
		Content:     event.Message.Content,
		IsCommand:   strings.HasPrefix(event.Message.Content, "!"),
		Timestamp:   event.Timestamp,
		ChannelName: event.Channel.Name,
		Me:          event.Message.Me,
		Highlighted: event.Message.Highlighted,
	}

	m.User.DisplayName = event.ChannelUser.User.DisplayName
	m.User.Username = event.ChannelUser.User.Username
	if event.ChannelUser.User.Color != "" {
		m.User.Color = event.ChannelUser.User.Color
	} else {
		// if a user does not have a color we hash his username
		// and take the first 6 chars as their color
		checksum := md5.Sum([]byte(m.User.Username))
		m.User.Color = "#" + hex.EncodeToString(checksum[:])[0:6]
	}

	m.User.IsPartner = event.ChannelUser.User.IsPartner
	m.User.IsMod = event.ChannelUser.IsMod
	m.User.IsVIP = event.ChannelUser.IsVIP
	m.User.IsSubscriber = event.ChannelUser.IsSubscriber
	m.User.IsBroadcaster = event.ChannelUser.IsBroadcaster
	m.User.SubscriberMonths = event.ChannelUser.SubscriberMonths
	m.User.SubscriberBadgeMonths, _ = strconv.ParseInt(event.ChannelUser.Badges["subscriber"], 10, 64)

	m.User.Badges = event.ChannelUser.Badges
	twitch.RLock()
	for name, version := range m.User.Badges {
		if name == "subscriber" {
			continue
		}
		m.User.BadgeURLs = append(m.User.BadgeURLs, twitch.globalBadges[name][version].ImageURL)
	}
	twitch.RUnlock()

	if _, ok := event.ChannelUser.Badges["founder"]; ok {
		m.User.IsFounder = true
	}

	// get user profile information
	if twitchUserDetails, err := twitch.getUser(m.User.Username); err != nil {
		log.Error(err)
	} else {
		m.User.LogoURL = twitchUserDetails.LogoURL
		m.User.ID = twitchUserDetails.ID
	}

	twitch.RLock()
	if badge, ok := twitch.subscriberBadges[m.User.SubscriberBadgeMonths]; ok {
		m.User.SubscriberBadgeURL = badge.ImageURL
	}
	twitch.RUnlock()

	for emoteID, ranges := range event.Message.Emotes {
		e := &TwitchEmote{ID: emoteID}

		for _, r := range ranges {
			e.Ranges = append(e.Ranges, struct {
				From int "json:\"from\""
				To   int "json:\"to\""
			}{
				From: r.From,
				To:   r.To,
			})
		}

		m.Emotes = append(m.Emotes, e)
	}

	// fetch nse data
	r, err := http.Get(strings.Trim(os.Getenv("STEVE_URL"), " /") + "/user/" + m.User.Username)
	if err == nil {
		body, err := ioutil.ReadAll(r.Body)
		if err == nil {
			// we ignore any error on this
			json.Unmarshal(body, &m.User)
		}
	}

	hugo.hub.broadcast(m)
}

func (twitch *Twitch) eventClearchat(t *twirgo.Twitch, event twirgo.EventClearchat) {
	hugo.hub.broadcast(TwitchClearchat{
		Username: event.User.Username,
	})
}

func (twitch *Twitch) eventClearmsg(t *twirgo.Twitch, event twirgo.EventClearmsg) {
	hugo.hub.broadcast(TwitchClearmsg{
		Username: event.User.Username,
		MsgID:    event.Message.ID,
	})
}
