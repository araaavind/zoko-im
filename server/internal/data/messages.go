package data

import (
	"context"
	"database/sql"
	"time"

	"github.com/araaavind/zoko-im/internal/validator"
)

type Message struct {
	ID         int64     `json:"id"`
	Timestamp  time.Time `json:"timestamp"`
	Content    string    `json:"content"`
	SenderID   int64     `json:"sender_id"`
	ReceiverID int64     `json:"receiver_id"`
	ReadStatus bool      `json:"read"`
}

type MessageModel struct {
	// DB *pgxpool.Pool
	DB *sql.DB
}

func ValidateMessage(v *validator.Validator, message *Message) {
	v.Check(message.Content != "", "content", "Content is required")
	v.Check(len(message.Content) <= 1000, "content", "Content must be less than 1000 characters")
}

func (m *MessageModel) Insert(message *Message) error {
	query := `
		INSERT INTO messages (timestamp, content, sender_id, receiver_id, read_status)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id`

	args := []any{
		message.Timestamp,
		message.Content,
		message.SenderID,
		message.ReceiverID,
		message.ReadStatus,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	return m.DB.QueryRowContext(ctx, query, args...).Scan(&message.ID)
}

func (m *MessageModel) GetAllForSenderReceiver(senderID int64, receiverID int64, filters Filters) ([]*Message, Metadata, error) {
	query := `
		SELECT id, timestamp, content, read_status, sender_id, receiver_id
		FROM messages
		WHERE ((sender_id = $1 AND receiver_id = $2) OR (sender_id = $2 AND receiver_id = $1))
		AND (timestamp < $3)
		ORDER BY timestamp DESC
		LIMIT $4
	`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	rows, err := m.DB.QueryContext(ctx, query, senderID, receiverID, filters.Cursor, filters.PageSize)
	if err != nil {
		return nil, getEmptyMetadata(filters.Cursor, filters.PageSize), err
	}
	defer rows.Close()

	messages := []*Message{}

	for rows.Next() {
		var message Message
		err := rows.Scan(&message.ID, &message.Timestamp, &message.Content, &message.ReadStatus, &message.SenderID, &message.ReceiverID)
		if err != nil {
			return nil, getEmptyMetadata(filters.Cursor, filters.PageSize), err
		}
		messages = append(messages, &message)
	}

	if err = rows.Err(); err != nil {
		return nil, getEmptyMetadata(filters.Cursor, filters.PageSize), err
	}

	var nextCursor time.Time

	if len(messages) > 0 {
		nextCursor = messages[len(messages)-1].Timestamp.UTC()
	}

	metadata := calculateMetadata(filters.Cursor, nextCursor, len(messages), filters.PageSize)

	return messages, metadata, nil
}
