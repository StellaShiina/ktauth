package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/StellaShiina/ktauth/internal/model"
)

type UserRepo struct {
	db *sql.DB
}

func NewUserRepo(db *sql.DB) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) NewUser(ctx context.Context, UUID, name, password_hash, email string) error {
	_, err := r.db.ExecContext(ctx, "INSERT INTO user (uuid, name, password_hash, email) VALUES (?, ?, ?, ?)", UUID, name, password_hash, email)
	return err
}

func (r *UserRepo) DelUser(ctx context.Context, name string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM user WHERE name = ?", name)
	return err
}

func (r *UserRepo) GetUserByName(ctx context.Context, name string) (model.User, error) {
	var user model.User
	row := r.db.QueryRowContext(ctx, "SELECT * FROM user WHERE name = ?", name)
	if err := row.Scan(&user.UUID, &user.Name, &user.PasswordHash, &user.Email); err != nil {
		if err == sql.ErrNoRows {
			return model.User{}, fmt.Errorf("No such user: %s %v", name, err)
		}
		return model.User{}, fmt.Errorf("Error when scanning: %v", err)
	}
	return user, nil
}

func (r *UserRepo) ListUsers(ctx context.Context) ([]model.User, error) {
	var users []model.User

	rows, err := r.db.QueryContext(ctx, "SELECT uuid, name FROM user")

	if err != nil {
		return nil, fmt.Errorf("Query error: %v", err)
	}

	defer rows.Close()

	for rows.Next() {
		var user model.User
		if err := rows.Scan(&user.UUID, &user.Name); err != nil {
			return nil, fmt.Errorf("Scan error: %v", err)
		}
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows.Err(): %v", err)
	}
	return users, nil
}
