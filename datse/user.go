package main

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
)

type User struct {
	Username         string `db:"username"`
	Status           string `db:"status"`
	Team             string `db:"team"`
	Taler            int    `db:"taler"`
	ReputationPoints int    `db:"reputation_points"`
}

// /user
func GETUsers(w http.ResponseWriter, r *http.Request) {
	users := []User{}
	err := db.Select(&users, "SELECT username, status, team, taler, reputation_points FROM users ORDER BY reputation_points DESC LIMIT 10")
	if err != nil {
		log.Error(err)
		return
	}

	json.NewEncoder(w).Encode(users)
}

// /user/{username}
func GETUser(w http.ResponseWriter, r *http.Request) {
	username := normalizeParameter(mux.Vars(r)["username"])

	if username == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	user := User{}
	err := db.Get(&user, "SELECT username, status, team, taler, reputation_points FROM users WHERE username = $1", username)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		log.Error(err)
		return
	}

	json.NewEncoder(w).Encode(user)
}

// /user/{username}
// /user/{username}/taler
// /user/{username}/reputation_points
func PUTUser(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	username := normalizeParameter(params["username"])
	subTarget := normalizeParameter(params["sub_target"])

	if username == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	switch subTarget {
	case "status":

	case "team":

	case "taler":

	case "reputation_points":

	case "":

	default:
		w.WriteHeader(http.StatusBadRequest)
	}
}
