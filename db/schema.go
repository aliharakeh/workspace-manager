package db

import (
	"database/sql"
	_ "embed"
)

// schemaSQL is the same file sqlc reads. Every statement is idempotent, so it
// runs on every start: it creates a fresh database and leaves an existing one
// untouched.
//
//go:embed schema.sql
var schemaSQL string

func applySchema(sqlDB *sql.DB) error {
	_, err := sqlDB.Exec(schemaSQL)
	return err
}
