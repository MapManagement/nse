package main

import (
	"net/http"
	"os"

	"github.com/sirupsen/logrus"
)

type ()

var (
	twitch *Twitch
	hugo   *Hugo
	cron   *Cron

	log = logrus.New()
)

func main() {
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "debug" {
		log.SetLevel(logrus.DebugLevel)
	} else if logLevel == "info" {
		log.SetLevel(logrus.InfoLevel)
	} else {
		log.SetLevel(logrus.ErrorLevel)
	}

	cron = newCron()

	// connect to twitch chat and register messageReceived method
	twitch = newTwitch()

	hugo = newHugo()

	http.HandleFunc("/ws", hugo.Serve)
	http.HandleFunc("/login", twitch.loginHandler)
	http.HandleFunc("/return", twitch.returnHandler)

	http.HandleFunc("/subcount", twitch.subcountHandler)

	log.Info("Listening on: ", os.Getenv("WS_PORT"))
	log.Fatal(http.ListenAndServe(":"+os.Getenv("WS_PORT"), nil))
}
