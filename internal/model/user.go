package model

import "database/sql"

type User struct {
	UUID         sql.NullString
	Name         string
	PasswordHash string
	Email        *string
}
