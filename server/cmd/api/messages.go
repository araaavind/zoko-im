package main

import (
	"net/http"
	"time"

	"github.com/araaavind/zoko-im/internal/data"
	"github.com/araaavind/zoko-im/internal/validator"
)

func (app *application) sendMessage(w http.ResponseWriter, r *http.Request) {
	senderID, err := app.readIDParam(r, "sender_id")
	if err != nil || senderID < 1 {
		app.notFoundResponse(w, r)
		return
	}

	receiverID, err := app.readIDParam(r, "receiver_id")
	if err != nil || receiverID < 1 {
		app.notFoundResponse(w, r)
		return
	}

	_, err = app.models.Users.Get(senderID)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	_, err = app.models.Users.Get(receiverID)
	if err != nil {
		app.notFoundResponse(w, r)
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
		SenderID:   senderID,
		ReceiverID: receiverID,
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

	err = app.writeJSON(w, http.StatusAccepted, envelope{"message": "Message queued for processing"}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
