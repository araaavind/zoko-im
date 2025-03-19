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

type Chat struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type MessageModel struct {
	// DB *pgxpool.Pool
	DB *sql.DB
}

func ValidateMessage(v *validator.Validator, message *Message) {
	v.Check(message.Content != "", "content", "Content is required")
	v.Check(len(message.Content) <= 1000, "content", "Content must be less than 1000 characters")
}

func (m *MessageModel) Insert(ctx context.Context, message *Message) error {
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

	return m.DB.QueryRowContext(ctx, query, args...).Scan(&message.ID)
}

// Inserts multiple messages in a single transaction
func (m *MessageModel) BulkInsert(ctx context.Context, messages []*Message) error {
	if len(messages) == 0 {
		return nil
	}

	tx, err := m.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `
		INSERT INTO messages (timestamp, content, sender_id, receiver_id, read_status)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id`

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, message := range messages {
		args := []any{
			message.Timestamp,
			message.Content,
			message.SenderID,
			message.ReceiverID,
			message.ReadStatus,
		}

		err = stmt.QueryRowContext(ctx, args...).Scan(&message.ID)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (m *MessageModel) GetAllForSenderReceiver(ctx context.Context, senderID int64, receiverID int64, filters Filters) ([]*Message, Metadata, error) {
	query := `
		SELECT id, timestamp, content, read_status, sender_id, receiver_id
		FROM messages
		WHERE ((sender_id = $1 AND receiver_id = $2) OR (sender_id = $2 AND receiver_id = $1))
		AND (timestamp < $3)
		ORDER BY timestamp DESC
		LIMIT $4
	`

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

func (m *MessageModel) UpdateStatus(ctx context.Context, messageID int64, readStatus bool) error {
	query := `
		UPDATE messages
		SET read_status = $1
		WHERE id = $2
	`

	res, err := m.DB.ExecContext(ctx, query, readStatus, messageID)
	if err != nil {
		return err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrRecordNotFound
	}
	return nil
}

func (m *MessageModel) GetAllChatsForUser(ctx context.Context, userID int64) ([]*Chat, error) {
	query := `
		SELECT DISTINCT messages.receiver_id, users.full_name
		FROM messages
		JOIN users ON messages.receiver_id = users.id
		WHERE sender_id = $1
	`

	rows, err := m.DB.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	chats := []*Chat{}

	for rows.Next() {
		var chat Chat
		err := rows.Scan(&chat.ID, &chat.Name)
		if err != nil {
			return nil, err
		}
		chats = append(chats, &chat)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return chats, nil
}
