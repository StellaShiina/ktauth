package model

type User struct {
	UUID         string
	Name         string
	PasswordHash string
	Email        *string
	Role         string
}
