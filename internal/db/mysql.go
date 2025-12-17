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

func NewMySQL(c Config) (*sql.DB, error) {
	cfg := mysql.NewConfig()
	cfg.User = c.User
	cfg.Passwd = c.Passwd
	cfg.Net = c.Net
	cfg.Addr = c.Addr
	cfg.DBName = c.DBName
	cfg.ParseTime = c.ParseTime
	cfg.Loc = &c.Loc

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
