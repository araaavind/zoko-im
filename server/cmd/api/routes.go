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
	router.HandlerFunc(http.MethodPost, "/v1/users/:user_id/chats/:peer_id/messages", app.sendMessage)
	router.HandlerFunc(http.MethodGet, "/v1/users/:user_id/chats/:peer_id/messages", app.listMessages)
	router.HandlerFunc(http.MethodGet, "/v1/users/:user_id/chats/:peer_id/subscribe", app.subscribe)
	router.HandlerFunc(http.MethodPatch, "/v1/messages/:message_id/read", app.readMessage)

	return app.recoverPanic(app.enableCORS(app.rateLimit(router)))
}
