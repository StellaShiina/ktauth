package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/StellaShiina/ktauth/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrUserNotFound = errors.New("user not found")
var ErrUserExist = errors.New("user already exist")

type UserRepo struct {
	pool *pgxpool.Pool
}

func NewUserRepo(pool *pgxpool.Pool) *UserRepo {
	return &UserRepo{pool: pool}
}

func (r *UserRepo) NewUser(ctx context.Context, UUID, name, password_hash string, email *string, role string) error {
	_, err := r.pool.Exec(ctx, "INSERT INTO users (uuid, name, password_hash, email, role) VALUES ($1, $2, $3, $4, $5)", UUID, name, password_hash, email, "user")
	if err != nil {
		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == "23505" {
			return ErrUserExist
		}
	}
	return err
}

func (r *UserRepo) DelUser(ctx context.Context, name string) error {
	cmdTag, err := r.pool.Exec(ctx, "DELETE FROM users WHERE name = $1", name)
	if cmdTag.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return err
}

func (r *UserRepo) GetUserByName(ctx context.Context, name string) (model.User, error) {
	var user model.User
	row := r.pool.QueryRow(ctx, "SELECT * FROM users WHERE name = $1", name)
	if err := row.Scan(&user.UUID, &user.Name, &user.PasswordHash, &user.Email, &user.Role); err != nil {
		if err == pgx.ErrNoRows {
			return model.User{}, fmt.Errorf("No such user: %s %w", name, err)
		}
		return model.User{}, fmt.Errorf("Error when scanning: %w", err)
	}
	return user, nil
}

func (r *UserRepo) ListUsers(ctx context.Context) ([]model.User, error) {
	var users []model.User

	rows, err := r.pool.Query(ctx, "SELECT uuid, name, email, role FROM users")

	if err != nil {
		return nil, fmt.Errorf("Query error: %w", err)
	}

	defer rows.Close()

	for rows.Next() {
		var user model.User
		if err := rows.Scan(&user.UUID, &user.Name, &user.Email, &user.Role); err != nil {
			return nil, fmt.Errorf("Scan error: %w", err)
		}
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows.Err(): %w", err)
	}

	return users, nil
}

func (r *UserRepo) UpdateUser(ctx context.Context, uuid, name, password_hash string, email *string, role string) error {
	commandTag, err := r.pool.Exec(ctx, "UPDATE users SET name = $1, password_hash = $2, email = $3, role = $4 WHERE uuid = $5", name, password_hash, email, role, uuid)
	if err != nil {
		return fmt.Errorf("Failed to update: %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return ErrUserNotFound
	}

	return nil
}
