package db

import (
	"database/sql"
	"time"

	"github.com/go-sql-driver/mysql"
)

type Config struct {
	User      string
	Passwd    string
	Net       string
	Addr      string
	DBName    string
	ParseTime bool
	Loc       time.Location
}

var config = Config{
	User:      "ktauth",
	Passwd:    "ktauth",
	Net:       "tcp",
	Addr:      "127.0.0.1",
	DBName:    "ktauth",
	ParseTime: true,
	Loc:       *time.Local,
}

func NewMySQL() (*sql.DB, error) {
	cfg := mysql.NewConfig()
	cfg.User = config.User
	cfg.Passwd = config.Passwd
	cfg.Net = config.Net
	cfg.Addr = config.Addr
	cfg.DBName = config.DBName
	cfg.ParseTime = config.ParseTime
	cfg.Loc = &config.Loc

	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		return nil, err
	}

	// db.SetMaxOpenConns(25)
	// db.SetMaxIdleConns(10)
	// db.SetConnMaxLifetime(5 * time.Minute)

	return db, nil
}
