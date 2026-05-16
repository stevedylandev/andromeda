// Package sqlite provides a shared SQLite bootstrap for andromeda Go apps.
package sqlite

import (
	"database/sql"

	_ "modernc.org/sqlite"
)

// Open opens a SQLite database at path with the connection settings and
// PRAGMAs used across andromeda Go apps. If schema is non-empty it is
// executed once after opening (typically CREATE TABLE IF NOT EXISTS ...).
func Open(path string, schema string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		db.Close()
		return nil, err
	}
	if schema != "" {
		if _, err := db.Exec(schema); err != nil {
			db.Close()
			return nil, err
		}
	}
	return db, nil
}
