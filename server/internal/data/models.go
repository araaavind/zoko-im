package data

import (
	"database/sql"
	"errors"
)

var ErrRecordNotFound = errors.New("record not found")

type Models struct {
	Messages MessageModel
	Users    UserModel
}

func NewModels(db *sql.DB) Models {
	return Models{
		Messages: MessageModel{DB: db},
		Users:    UserModel{DB: db},
	}
}

// func NewModelsWithPGX(db *pgxpool.Pool) Models {
// 	return Models{
// 		Messages: MessageModel{DB: db},
// 		Users:    UserModel{DB: db},
// 	}
// }
