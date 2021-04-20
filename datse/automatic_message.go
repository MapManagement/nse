package main

import "net/http"

type AutomaticMessage struct {
	Interval int    `db:"interval"`
	Active   bool   `db:"active"`
	Content  string `db:"content"`
}

// /automatic_message
func GETAutomaticMessages(w http.ResponseWriter, r *http.Request) {

}
