package main

import "net/http"

type Command struct {
	Name  string `db:"name"`
	Value string `db:"value"`
}

// /command
func GETCommands(w http.ResponseWriter, r *http.Request) {

}

// /command/{name}
func GETCommand(w http.ResponseWriter, r *http.Request) {

}

// /command/{name}
func PUTCommand(w http.ResponseWriter, r *http.Request) {

}
