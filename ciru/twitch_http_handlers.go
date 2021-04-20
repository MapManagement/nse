package main

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

func (twitch *Twitch) loginHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, twitch.oauthConfig.AuthCodeURL("12345"), http.StatusTemporaryRedirect)
}

func (twitch *Twitch) returnHandler(w http.ResponseWriter, r *http.Request) {
	var err error
	twitch.oauthToken, err = twitch.oauthConfig.Exchange(context.Background(), r.URL.Query().Get("code"))
	if err != nil {
		log.Error(err)
		return
	}

	cron.Stop("refresh_oauth_token")
	cron.New("refresh_oauth_token", twitch.refreshAccessToken, 3000*time.Second)
}

func (twitch *Twitch) subcountHandler(w http.ResponseWriter, r *http.Request) {
	subCount, err := twitch.getBroadcasterSubscriptions()
	res := struct {
		SubCount int   `json:"subcount"`
		Error    error `json:"error"`
	}{
		SubCount: subCount,
		Error:    err,
	}

	resBody, err := json.Marshal(res)
	if err != nil || res.Error != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNotImplemented)
	w.Write(resBody)
}
