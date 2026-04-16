package db

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Config struct {
	User   string
	Passwd string
	Host   string
	Port   string
	DBName string
}

var config = Config{
	User:   "ktauth",
	Passwd: "ktauth",
	Host:   "127.0.0.1",
	Port:   "5432",
	DBName: "ktauth",
}

func NewPostgres() (*pgxpool.Pool, error) {
	if envHost := os.Getenv("POSTGRES_HOST"); envHost != "" {
		config.Host = envHost
	}
	if envPort := os.Getenv("POSTGRES_PORT"); envPort != "" {
		config.Port = envPort
	}

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", config.User, config.Passwd, config.Host, config.Port, config.DBName)

	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}

	// 连接池
	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		return nil, err
	}

	// 测试连接
	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		return nil, err
	}

	return pool, nil
}
