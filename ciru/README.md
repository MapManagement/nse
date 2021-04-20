# Twitch chat service aka Hugo

## Env vars

| Name                | Description                                                  |
| ------------------- | ------------------------------------------------------------ |
| WS_PORT             | Websocket port                                               |
| LOG_LEVEL           | info (default: error)                                        |
| TWITCH_USERNAME     | Username of the Twitch bot account                           |
| TWITCH_CHANNEL      | Twitch channel                                               |
| TWITCH_TOKEN        | Twitch oauth token                                           |
| TWITCH_CLIENTID     | Client ID for Twitch api requests                            |
| TWITCH_CLIENTSECRET | Client Secret for Twitch api requests                        |
| STEVE_URL           | URL of the data service steve                                |
| BASE_URL            | Base URL for hugo which is used for the Twitch OAuth process |
