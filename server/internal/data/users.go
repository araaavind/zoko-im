package data

import (
	"context"
	"database/sql"
	"errors"
)

type User struct {
	ID       int64  `json:"id"`
	FullName string `json:"full_name"`
}

type UserModel struct {
	DB *sql.DB
}

func (m UserModel) Get(ctx context.Context, id int64) (*User, error) {
	if id < 1 {
		return nil, ErrRecordNotFound
	}

	query := `
	SELECT id, full_name
	FROM users
	WHERE id = $1
	`

	var user User

	err := m.DB.QueryRowContext(ctx, query, id).Scan(&user.ID, &user.FullName)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}

	return &user, nil
}
