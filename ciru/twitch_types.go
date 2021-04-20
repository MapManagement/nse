package main

import (
	"net/http"
	"sync"
	"time"

	"github.com/curiTTV/twirgo"
	"github.com/gorilla/websocket"
	"golang.org/x/oauth2"
)

type (
	Twitch struct {
		*sync.RWMutex

		twirgo            *twirgo.Twitch
		pubSub            *TwitchPubSub
		automaticMessages *TwitchAutomaticMessages

		channelID       string
		clientID        string
		httpClient      *http.Client
		oAuthHTTPClient *http.Client

		users map[string]*TwitchUserDetails

		bitsBadges       map[int64]*TwitchBadge
		subscriberBadges map[int64]*TwitchBadge
		globalBadges     map[string]map[string]*TwitchBadge

		oauthToken  *oauth2.Token
		oauthConfig *oauth2.Config

		isOnline bool
	}

	TwitchMessage struct {
		ID          string         `json:"id"`
		Timestamp   time.Time      `json:"timestamp"`
		Content     string         `json:"content"`
		IsCommand   bool           `json:"isCommand"`
		Emotes      []*TwitchEmote `json:"emotes"`
		ChannelName string         `json:"channelName"`
		Highlighted bool           `json:"highlighted"`
		Me          bool           `json:"me"`
		User        struct {
			ID                    string `json:"id"`
			DisplayName           string `json:"displayName"`
			Username              string `json:"username"`
			Color                 string `json:"color"`
			IsPartner             bool   `json:"isPartner"`
			IsFounder             bool   `json:"isFounder"`
			IsMod                 bool   `json:"isMod"`
			IsVIP                 bool   `json:"isVIP"`
			IsSubscriber          bool   `json:"isSubscriber"`
			IsBroadcaster         bool   `json:"isBroadcaster"`
			SubscriberMonths      int64  `json:"subscriberMonths"`
			SubscriberBadgeMonths int64  `json:"subscriberBadgeMonths"`
			SubscriberBadgeURL    string `json:"subscriberBadgeURL"`
			// key: name
			// value: amount
			Badges           map[string]string `json:"badges"`
			BadgeURLs        []string          `json:"badgeURLs"`
			LogoURL          string            `json:"logoURL"`
			Taler            int               `json:"taler"`
			Status           string            `json:"status"`
			Team             string            `json:"team"`
			ReputationPoints int               `json:"reputationPoints"`
		} `json:"user"`
	}

	TwitchClearchat struct {
		Username string `json:"username"`
	}

	TwitchClearmsg struct {
		Username string `json:"username"`
		MsgID    string `json:"msgID"`
	}

	TwitchUserDetails struct {
		ID               string `json:"_id"`
		Username         string `json:"name"`
		LogoURL          string `json:"logo"`
		fetchedTimestamp time.Time
	}

	TwitchBadge struct {
		ImageURL string `json:"image_url_4x"`
	}

	TwitchEmote struct {
		ID     string `json:"id"`
		Ranges []struct {
			From int `json:"from"`
			To   int `json:"to"`
		} `json:"ranges"`
	}

	TwitchPubSub struct {
		conn *websocket.Conn

		closeConn     chan bool
		writeMessages chan *TwitchPubSubRequest

		lastPing time.Time

		readListenerClosed  bool
		writeListenerClosed bool
		listenMessageSent   bool
	}

	TwitchPubSubRequest struct {
		Type  string                   `json:"type"`
		Nonce string                   `json:"nonce,omitempty"`
		Data  *TwitchPubSubRequestData `json:"data,omitempty"`
	}

	TwitchPubSubRequestData struct {
		Topics    []string `json:"topics"`
		AuthToken string   `json:"auth_token"`
	}

	TwitchPubSubResponse struct {
		Type  string `json:"type,omitempty"`
		Nonce string `json:"nonce,omitempty"`
		Error string `json:"error,omitempty"`
		Data  struct {
			Topic   string `json:"topic"`
			Message string `json:"message"`
		} `json:"data"`
	}

	TwitchPubSubMessageReward struct {
		Type string `json:"type"`
		Data struct {
			Timestamp  time.Time `json:"timestamp,omitempty"`
			Redemption struct {
				ID   string `json:"id"`
				User struct {
					ID          string `json:"id,omitempty"`
					Login       string `json:"login,omitempty"`
					DisplayName string `json:"display_name,omitempty"`
				} `json:"user"`
				ChannelID  string    `json:"channel_id,omitmepty"`
				RedeemedAt time.Time `json:"redemmed_at,omitempty"`
				Reward     struct {
					ID                  string `json:"id,omitempty"`
					ChannelID           string `json:"channel_id,omitempty"`
					Title               string `json:"title,omitempty"`
					Prompt              string `json:"prompt,omitempty"`
					Cost                int    `json:"cost,omitempty"`
					IsUserInputRequired bool   `json:"is_user_input_required,omitempty"`
					IsSubOnly           bool   `json:"is_sub_only,omitempty"`
					Image               struct {
						URL1x string `json:"url_1x,omitempty"`
						URL2x string `json:"url_2x,omitempty"`
						URL4x string `json:"url_4x,omitempty"`
					} `json:"image"`
					DefaultImage struct {
						URL1x string `json:"url_1x,omitempty"`
						URL2x string `json:"url_2x,omitempty"`
						URL4x string `json:"url_4x,omitempty"`
					} `json:"default_image"`
					BackgroundColor string `json:"background_color,omitempty"`
					IsEnabled       bool   `json:"is_enabled,omitempty"`
					IsPaused        bool   `json:"is_paused,omitempty"`
					IsInStock       bool   `json:"is_in_stock,omitempty"`
					MaxPerStream    struct {
						IsEnabled    bool `json:"is_enabled,omitempty"`
						MaxPerStream int  `json:"max_per_stream,omitempty"`
					} `json:"max_per_stream"`
					ShouldRedemptionsSkipRequestQueue bool `json:"should_redemptions_skip_request_queue,omitempty"`
				} `json:"reward"`
				UserInput string `json:"user_input,omitempty"`
				Status    string `json:"status,omitempty"`
			} `json:"redemption"`
		} `json:"data"`
	}

	TwitchPubSubMessageSub struct {
		Username         string    `json:"user_name,omitempty"`
		DisplayName      string    `json:"display_name,omitempty"`
		ChannelName      string    `json:"channel_name,omitempty"`
		UserID           string    `json:"user_id,omitempty"`
		ChannelID        string    `json:"channel_id,omitempty"`
		Time             time.Time `json:"time,omitempty"`
		SubPlan          string    `json:"sub_plan,omitempty"`
		SubPlanName      string    `json:"sub_plan_name,omitempty"`
		Months           int       `json:"months,omitempty"`
		CumulativeMonths int       `json:"cumulative_months,omitempty"`
		StreakMonths     int       `json:"streak_months,omitempty"`
		Context          string    `json:"context,omitempty"`
		IsGift           bool      `json:"is_gift,omitempty"`
		SubMessage       struct {
			Message string `json:"message,omitempty"`
			Emotes  []struct {
				Start int `json:"start,omitempty"`
				End   int `json:"end,omitempty"`
				ID    int `json:"id,omitempty"`
			} `json:"emotes,omitempty"`
		} `json:"sub_message,omitempty"`
		RecipientID          string `json:"recipient_id,omitempty"`
		RecipientUserName    string `json:"recipient_user_name,omitempty"`
		RecipientDisplayName string `json:"recipient_display_name,omitempty"`
		MultiMonthDuration   int    `json:"multi_month_duration,omitempty"`
	}

	TwitchPubSubMessageCheer struct {
		Data struct {
			Username         string    `json:"user_name"`
			ChannelName      string    `json:"channel_name"`
			UserID           string    `json:"user_id"`
			ChannelID        string    `json:"channel_id"`
			Time             time.Time `json:"time"`
			ChatMessage      string    `json:"chat_message"`
			BitsUsed         int       `json:"bits_used"`
			TotalBitsUsed    int       `json:"total_bits_used"`
			Context          string    `json:"context"`
			BadgeEntitlement struct {
				NewVersion      int `json:"new_version"`
				PreviousVersion int `json:"previous_version"`
			} `json:"badge_entitlement"`
		} `json:"data"`
		Version     string `json:"version"`
		MessageType string `json:"message_type"`
		MessageID   string `json:"message_id"`
		IsAnonymous bool   `json:"is_anonymous"`
	}

	TwitchAutomaticMessages struct {
		*sync.RWMutex

		messages []*TwitchAutomaticMessage

		// key: id
		// value: next send time
		scheduledMessages map[int]time.Time
	}

	TwitchAutomaticMessage struct {
		ID       int    `json:"id"`
		Interval int    `json:"interval"`
		Content  string `json:"content"`
	}
)
