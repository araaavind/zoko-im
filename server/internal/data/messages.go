package data

import (
	"context"
	"database/sql"
	"time"
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
