package main

import (
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"
)

var (
	log *logrus.Logger
	db  *sqlx.DB
)

func main() {
	log = logrus.New()
	log.Level = logrus.InfoLevel

	var err error

	db, err = sqlx.Connect("postgres", "host="+os.Getenv("NSE_DB_HOST")+" port="+os.Getenv("NSE_DB_PORT")+" user="+os.Getenv("NSE_DB_USER")+" password="+os.Getenv("NSE_DB_PASS")+" dbname="+os.Getenv("NSE_DB_NAME")+" sslmode=disable TimeZone="+os.Getenv("NSE_TIMEZONE"))
	if err != nil {
		log.Panic(err)
	}

	srv := &http.Server{
		Addr:         "0.0.0.0:8080",
		ReadTimeout:  time.Second * 30,
		WriteTimeout: time.Second * 30,
		IdleTimeout:  time.Second * 30,
	}

	r := mux.NewRouter()

	// user endpoints
	r.HandleFunc("/user", GETUsers).Methods("GET")
	r.HandleFunc("/user/{username}", GETUser).Methods("GET")
	r.HandleFunc("/user/{username}", PUTUser).Methods("PUT")
	r.HandleFunc("/user/{username}/taler", PUTUser).Methods("PUT")
	r.HandleFunc("/user/{username}/reputation_points", PUTUser).Methods("PUT")

	// command endpoints
	r.HandleFunc("/command", GETCommands).Methods("GET")
	r.HandleFunc("/command/{name}", GETCommand).Methods("GET")
	r.HandleFunc("/command/{name}", PUTCommand).Methods("PUT")
	r.HandleFunc("/command/{name}/{sub_target}", PUTCommand).Methods("PUT")

	// automatic message(s) endpoints
	r.HandleFunc("/automatic_message", GETAutomaticMessages).Methods("GET")

	srv.Handler = r

	if err := srv.ListenAndServe(); err != nil {
		log.Panic(err)
	}
}

func normalizeParameter(username string) string {
	return strings.TrimSpace(strings.ToLower(username))
}
