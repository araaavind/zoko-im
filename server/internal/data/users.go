package data

import (
	"context"
	"database/sql"
	"time"
)

type User struct {
	ID       int64  `json:"id"`
	FullName string `json:"full_name"`
}

type UserModel struct {
	DB *sql.DB
}

func (m UserModel) Get(id int64) (*User, error) {
	if id < 1 {
		return nil, ErrRecordNotFound
	}

	query := `
	SELECT id, full_name
	FROM users
	WHERE id = $1
	`

	var user User

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := m.DB.QueryRowContext(ctx, query, id).Scan(&user.ID, &user.FullName)
	if err != nil {
		return nil, err
	}

	return &user, nil
}
