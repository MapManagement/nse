package main

import (
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"golang.org/x/oauth2"
)

func (twitch *Twitch) fetchUser(username string) (*TwitchUserDetails, error) {
	body, err := twitch.apiRequest(twitch.httpClient, http.MethodGet, "https://api.twitch.tv/kraken/users?login="+username, nil, true)
	if err != nil {
		return nil, err
	}

	var respJSON struct {
		Users []*TwitchUserDetails `json:"users"`
	}
	err = json.Unmarshal(body, &respJSON)
	if err != nil {
		return nil, err
	}

	for _, user := range respJSON.Users {
		if username == user.Username {
			user.fetchedTimestamp = time.Now()
			twitch.Lock()
			twitch.users[user.Username] = user
			twitch.Unlock()
			return user, nil
		}
	}

	return nil, errors.New("User not found")
}

func (twitch *Twitch) apiRequest(httpClient *http.Client, method string, url string, body io.Reader, v5 bool) ([]byte, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	// it works for this cases but normally you would not send your accesstoken every time
	// if you do not query the v5 api
	if v5 {
		req.Header.Add("Accept", "application/vnd.twitchtv.v5+json")
	} else {
		req.Header.Add("Authorization", "Bearer "+twitch.oauthToken.AccessToken)
	}
	req.Header.Add("Client-ID", twitch.clientID)

	res, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	resBody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	log.Debugf("API Request %s %s: %s", method, url, body)

	return resBody, nil
}

func (twitch *Twitch) fetchChannelBadges() {
	res, err := http.Get("https://badges.twitch.tv/v1/badges/channels/" + twitch.channelID + "/display")
	if err != nil {
		log.Error("Could not get badges from Twitch: ", err)
		return
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Error("Channel badges: ", err)
		return
	}

	log.Debugf("fetchChannelBadges: %s", body)

	var respJSON struct {
		BadgeSets struct {
			Bits struct {
				Versions map[int64]*TwitchBadge `json:"versions"`
			} `json:"bits"`
			Subscriber struct {
				Versions map[int64]*TwitchBadge `json:"versions"`
			} `json:"subscriber"`
		} `json:"badge_sets"`
	}

	err = json.Unmarshal(body, &respJSON)
	if err != nil {
		log.Error("Channel badges: ", err)
		return
	}

	twitch.Lock()
	defer twitch.Unlock()
	twitch.bitsBadges = respJSON.BadgeSets.Bits.Versions
	twitch.subscriberBadges = respJSON.BadgeSets.Subscriber.Versions
}

func (twitch *Twitch) fetchGlobalBadges() {
	res, err := http.Get("https://badges.twitch.tv/v1/badges/global/display")
	if err != nil {
		log.Error("Could not get global badges from Twitch: ", err)
		return
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Error("Global badges: ", err)
		return
	}

	log.Debugf("fetchGlobalBadges: %s", body)

	var badges struct {
		BadgeSets map[string]struct {
			Versions map[string]*TwitchBadge `json:"versions"`
		} `json:"badge_sets"`
	}

	err = json.Unmarshal(body, &badges)
	if err != nil {
		log.Error("Global badges: ", err)
		return
	}

	twitch.Lock()
	twitch.globalBadges = make(map[string]map[string]*TwitchBadge)
	for name, b := range badges.BadgeSets {
		twitch.globalBadges[name] = b.Versions
	}
	defer twitch.Unlock()
}

func (twitch *Twitch) checkIfOnline() {
	body, err := twitch.apiRequest(twitch.httpClient, http.MethodGet, "https://api.twitch.tv/kraken/streams/"+twitch.channelID, nil, true)
	if err != nil {
		log.Error("Check if online: ", err)
		return
	}

	log.Debugf("checkIfOnline: %s", body)

	var res struct {
		Stream struct {
			ID          int64     `json:"_id"`
			Game        string    `json:"game"`
			Viewers     int       `json:"viewers"`
			VideoHeight int       `json:"video_height"`
			AverageFPS  int       `json:"average_fps"`
			Delay       int       `json:"delay"`
			CreatedAt   time.Time `json:"created_at"`
			IsPlaylist  bool      `json:"is_playlist"`
			Preview     struct {
				Small    string `json:"small"`
				Medium   string `json:"medium"`
				Large    string `json:"large"`
				Template string `json:"template"`
			} `json:"preview"`
			Channel struct {
				Mature                       bool      `json:"mature"`
				Status                       string    `json:"status"`
				BroadcasterLanguage          string    `json:"broadcaster_language"`
				DisplayName                  string    `json:"display_name"`
				Game                         string    `json:"game"`
				Languange                    string    `json:"language"`
				ID                           int64     `json:"_id"`
				Name                         string    `json:"name"`
				CreatedAt                    time.Time `json:"created_at"`
				UpdatedAt                    time.Time `json:"updated_at"`
				Partner                      bool      `json:"partner"`
				Logo                         string    `json:"logo"`
				VideoBanner                  string    `json:"video_banner"`
				ProfileBanner                string    `json:"profile_banner"`
				ProfileBannerBackgroundColor string    `json:"profile_banner_background_color"`
				URL                          string    `json:"url"`
				Views                        int64     `json:"views"`
				Followers                    int64     `json:"followers"`
			} `json:"channel"`
		} `json:"stream"`
	}
	err = json.Unmarshal(body, &res)
	if err != nil {
		log.Error("Check if online unmarshal: ", err)
		return
	}

	if res.Stream.ID > 0 {
		twitch.isOnline = true
	} else {
		twitch.isOnline = false
	}
}

func (twitch *Twitch) refreshAccessToken() {
	log.Info("Refreshing access token")
	tokenURL := "https://id.twitch.tv/oauth2/token"
	bodyString := "grant_type=refresh_token&refresh_token=" + twitch.oauthToken.RefreshToken + "&client_id=" + os.Getenv("TWITCH_CLIENTID") + "&client_secret=" + os.Getenv("TWITCH_CLIENTSECRET")
	body := strings.NewReader(bodyString)
	log.Debug("Sending Twitch API request to ", tokenURL, " with: ", bodyString)

	r, err := http.Post(tokenURL, "", body)
	if err != nil {
		log.Error("Could not refresh auth token: ", err)
		twitch.Lock()
		twitch.oauthToken = &oauth2.Token{}
		twitch.Unlock()
		return
	}

	var token oauth2.Token

	rBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Error("Can not read Twitch response for token refresh: ", err)
		twitch.Lock()
		twitch.oauthToken = &oauth2.Token{}
		twitch.Unlock()
		return
	}

	log.Debugf("Received response for access token refresh: %s", rBody)

	err = json.Unmarshal(rBody, &token)
	if err != nil {
		log.Error("Can not unmarshal Twitch response for token refresh: ", err)
		twitch.Lock()
		twitch.oauthToken = &oauth2.Token{}
		twitch.Unlock()
		return
	}

	twitch.Lock()
	log.Debug("Set new access and refresh token: ", token.AccessToken, " - ", token.RefreshToken)
	twitch.oauthToken = &token
	twitch.Unlock()
}

func (twitch *Twitch) getBroadcasterSubscriptions() (int, error) {
	body, err := twitch.apiRequest(twitch.httpClient, http.MethodGet, "https://api.twitch.tv/helix/subscriptions?broadcaster_id="+twitch.channelID, nil, false)
	if err != nil {
		log.Error("Broadcaster subscriptions: ", err)
		return 0, err
	}

	log.Debugf("Broadcaster subscriptions: %s", body)

	var res struct {
		Data []struct {
			BroadcasterID   string `json:"broadcaster_id"`
			BroadcasterName string `json:"broadcaster_name"`
			IsGift          bool   `json:"is_gift"`
			Tier            string `json:"tier"`
			PlanName        string `json:"plan_name"`
			UserID          string `json:"user_id"`
			Username        string `json:"user_name"`
		} `json:"data"`
	}

	err = json.Unmarshal(body, &res)
	if err != nil {
		log.Error("Broadcaster subscriptions unmarshal: ", err)
		return 0, err
	}

	// not completely implemented because Twitch does not deliver any endpoint with just the subcount
	return 0, nil
}
