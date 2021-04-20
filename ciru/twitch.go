package main

import (
	"context"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/curiTTV/twirgo"
	"golang.org/x/oauth2"
)

func newTwitch() *Twitch {
	log.Info("Init Twitch")
	twitch := &Twitch{
		oauthToken: &oauth2.Token{},
		oauthConfig: &oauth2.Config{
			ClientID:     os.Getenv("TWITCH_CLIENTID"),
			ClientSecret: os.Getenv("TWITCH_CLIENTSECRET"),
			Scopes:       []string{"channel:read:redemptions", "channel_subscriptions", "bits:read", "channel:read:subscriptions"},
			RedirectURL:  strings.TrimRight(os.Getenv("BASE_URL"), "/") + "/return",
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://id.twitch.tv/oauth2/authorize",
				TokenURL: "https://id.twitch.tv/oauth2/token",
			},
		},
	}

	twitch.Init()
	twitch.pubSub = newTwitchPubSub()
	twitch.automaticMessages = newAutomaticMessages()

	return twitch
}

func (twitch *Twitch) Init() {
	twitch.RWMutex = &sync.RWMutex{}
	twitch.httpClient = &http.Client{
		Timeout: 3 * time.Second,
	}
	twitch.oAuthHTTPClient = twitch.oauthConfig.Client(context.Background(), twitch.oauthToken)
	twitch.oAuthHTTPClient.Timeout = 3 * time.Second
	twitch.users = make(map[string]*TwitchUserDetails)

	options := twirgo.Options{
		Username:       os.Getenv("TWITCH_USERNAME"),
		Token:          os.Getenv("TWITCH_TOKEN"),
		Channels:       []string{strings.ToLower(os.Getenv("TWITCH_CHANNEL"))},
		Log:            log,
		DefaultChannel: os.Getenv("TWITCH_CHANNEL"),
	}

	twitch.clientID = os.Getenv("TWITCH_CLIENTID")
	if twitch.clientID == "" {
		log.Fatal("Env var TWITCH_CLIENTID is not set")
	}

	user, err := twitch.getUser(options.DefaultChannel)
	if err != nil {
		log.Error(err)
		log.Fatal("Could not get user information from Twitch")
	}

	twitch.channelID = user.ID

	twitch.fetchChannelBadges()
	twitch.fetchGlobalBadges()
	twitch.checkIfOnline()
	cron.New("channel_badges", twitch.fetchChannelBadges, 24*time.Hour)
	cron.New("global_badges", twitch.fetchGlobalBadges, 24*time.Hour)
	cron.New("check_if_online", twitch.checkIfOnline, 15*time.Minute)
	cron.New("clean_users", twitch.cleanUsers, 15*time.Minute)

	// initiate TWIRGO

	twitch.twirgo = twirgo.New(options)

	ch, err := twitch.twirgo.Connect()
	if err != nil {
		log.Fatal(err)
	}

	twitch.twirgo.OnMessageReceived(twitch.eventMessageReceived)
	twitch.twirgo.OnClearchat(twitch.eventClearchat)
	twitch.twirgo.OnClearmsg(twitch.eventClearmsg)

	go twitch.twirgo.Run(ch)
}

func (twitch *Twitch) getUser(username string) (*TwitchUserDetails, error) {
	username = strings.ToLower(strings.TrimSpace(username))

	if user, ok := twitch.users[username]; ok {
		return user, nil
	}

	return twitch.fetchUser(username)
}

func (twitch *Twitch) cleanUsers() {
	twitch.RLock()
	users := twitch.users
	twitch.RUnlock()

	for username := range users {
		if time.Now().After(twitch.users[username].fetchedTimestamp.Add(15 * time.Minute)) {
			twitch.Lock()
			delete(users, username)
			twitch.Unlock()
		}
	}
}
