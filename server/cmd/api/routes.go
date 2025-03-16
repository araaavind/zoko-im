package main

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func (app *application) routes() http.Handler {
	router := httprouter.New()

	router.NotFound = http.HandlerFunc(app.notFoundResponse)
	router.MethodNotAllowed = http.HandlerFunc(app.methodNotAllowedResponse)

	router.HandlerFunc(http.MethodGet, "/v1/healthcheck", app.healthcheck)
	router.HandlerFunc(http.MethodPost, "/v1/users/:sender_id/chats/:receiver_id/messages", app.sendMessage)
	router.HandlerFunc(http.MethodGet, "/v1/users/:sender_id/chats/:receiver_id/messages", app.listMessages)

	return router
}
