package main

import (
	"intercom"
	"log"
	"net/http"
)

func main() {
	server := intercom.NewServer(":80")

	// Define websocket connect url, default "/ws"
	server.WSPath = "/ws"
	// Define push message url, default "/push"
	server.PushPath = "/push"

	// Set AuthToken func to authorize websocket connection, token is sent by
	// client for registe.
	server.AuthToken = func(token string) (userID string, ok bool) {
		// TODO: check if token is valid and calculate userID
		return token, true
	}

	// Set PushAuth func to check push request. If the request is valid, returns
	// true. Otherwise return false and request will be ignored.
	server.PushAuth = func(r *http.Request) bool {
		// TODO: check if request is valid

		return true
	}

	// Run server
	log.Println("The intercom server is read for listening")

	if err := server.ListenAndServe(); err != nil {
		panic(err)
	}
}
