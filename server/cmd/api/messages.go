package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/araaavind/zoko-im/internal/data"
	"github.com/araaavind/zoko-im/internal/validator"
	"github.com/coder/websocket"
)

func (app *application) sendMessage(w http.ResponseWriter, r *http.Request) {
	userID, err := app.readIDParam(r, "user_id")
	if err != nil || userID < 1 {
		app.notFoundResponse(w, r)
		return
	}

	peerID, err := app.readIDParam(r, "peer_id")
	if err != nil || peerID < 1 {
		app.notFoundResponse(w, r)
		return
	}

	ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), time.Second)
	defer cancel()

	_, err = app.models.Users.Get(ctx, userID)
	if err != nil {
		if errors.Is(err, data.ErrRecordNotFound) {
			app.notFoundResponse(w, r)
		} else {
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	_, err = app.models.Users.Get(ctx, peerID)
	if err != nil {
		if errors.Is(err, data.ErrRecordNotFound) {
			app.notFoundResponse(w, r)
		} else {
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	var input struct {
		Content string `json:"content"`
	}

	err = app.readJSON(w, r, &input)
	if err != nil {
		app.errorResponse(w, r, http.StatusBadRequest, err.Error())
		return
	}

	v := validator.New()

	message := &data.Message{
		Timestamp:  time.Now(),
		Content:    input.Content,
		SenderID:   userID,
		ReceiverID: peerID,
		ReadStatus: false,
	}

	if data.ValidateMessage(v, message); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	err = app.queue.EnqueueMessage(r.Context(), message)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// Convert the message to JSON before publishing
	messageJSON, err := json.Marshal(message)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	app.hub.PublishToUser(userID, peerID, messageJSON)

	err = app.writeJSON(w, http.StatusAccepted, envelope{"message": "Message queued for processing"}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) listMessages(w http.ResponseWriter, r *http.Request) {
	userID, err := app.readIDParam(r, "user_id")
	if err != nil || userID < 1 {
		app.notFoundResponse(w, r)
		return
	}

	peerID, err := app.readIDParam(r, "peer_id")
	if err != nil || peerID < 1 {
		app.notFoundResponse(w, r)
		return
	}

	v := validator.New()
	qs := r.URL.Query()

	var filters data.Filters

	filters.Cursor = app.readTime(qs, "cursor", time.Now(), v)
	filters.PageSize = app.readInt(qs, "page_size", 20, v)

	if data.ValidateFilters(v, filters); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), time.Second)
	defer cancel()

	messages, metadata, err := app.models.Messages.GetAllForSenderReceiver(ctx, userID, peerID, filters)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"messages": messages, "metadata": metadata}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) readMessage(w http.ResponseWriter, r *http.Request) {
	messageID, err := app.readIDParam(r, "message_id")
	if err != nil || messageID < 1 {
		app.notFoundResponse(w, r)
		return
	}

	ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), time.Second)
	defer cancel()

	err = app.models.Messages.UpdateStatus(ctx, messageID, true)
	if err != nil {
		if errors.Is(err, data.ErrRecordNotFound) {
			app.notFoundResponse(w, r)
		} else {
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"status": "read"}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) subscribe(w http.ResponseWriter, r *http.Request) {
	userID, err := app.readIDParam(r, "user_id")
	if err != nil || userID < 1 {
		app.notFoundResponse(w, r)
		return
	}

	peerID, err := app.readIDParam(r, "peer_id")
	if err != nil || peerID < 1 {
		app.notFoundResponse(w, r)
		return
	}

	ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), time.Second)
	defer cancel()

	_, err = app.models.Users.Get(ctx, userID)
	if err != nil {
		if errors.Is(err, data.ErrRecordNotFound) {
			app.notFoundResponse(w, r)
		} else {
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	_, err = app.models.Users.Get(ctx, peerID)
	if err != nil {
		if errors.Is(err, data.ErrRecordNotFound) {
			app.notFoundResponse(w, r)
		} else {
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	app.hub.HandleConnection(r.Context(), userID, peerID, conn)
}
