package main

import (
	"database/sql"

	"github.com/stevedylandev/andromeda/pkg/sqlite"
)

func openDB(path string) (*sql.DB, error) {
	return sqlite.Open(path, "")
}
