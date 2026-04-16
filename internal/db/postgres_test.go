package db_test

import (
	"testing"

	"github.com/StellaShiina/ktauth/internal/db"
)

func TestPostgres(t *testing.T) {
	postgres, err := db.NewPostgres()
	if err != nil {
		t.Fatal(err)
	}

	if err := postgres.Ping(t.Context()); err != nil {
		t.Fatal(err)
	}
}
