package store

import (
	"database/sql"
	"errors"

	"github.com/stevedylandev/andromeda/crates-go/auth"
	_ "modernc.org/sqlite"
)

type Note struct {
	ID        int64  `json:"id"`
	ShortID   string `json:"short_id"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type NoteInput struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

const noteColumns = `id, short_id, title, content, created_at, updated_at`

const schema = `
CREATE TABLE IF NOT EXISTS notes (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    short_id   TEXT NOT NULL UNIQUE,
    title      TEXT NOT NULL,
    content    TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);
`

func Open(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return nil, err
	}
	if _, err := db.Exec(schema); err != nil {
		return nil, err
	}
	return db, nil
}

func scanNote(scanner interface{ Scan(dest ...any) error }) (*Note, error) {
	var n Note
	err := scanner.Scan(&n.ID, &n.ShortID, &n.Title, &n.Content, &n.CreatedAt, &n.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &n, nil
}

func Create(db *sql.DB, title, content string) (*Note, error) {
	shortID, err := auth.GenerateShortID(10)
	if err != nil {
		return nil, err
	}
	res, err := db.Exec(`INSERT INTO notes (short_id, title, content) VALUES (?, ?, ?)`, shortID, title, content)
	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	return scanNote(db.QueryRow(`SELECT `+noteColumns+` FROM notes WHERE id = ?`, id))
}

func GetByShortID(db *sql.DB, shortID string) (*Note, error) {
	return scanNote(db.QueryRow(`SELECT `+noteColumns+` FROM notes WHERE short_id = ?`, shortID))
}

func List(db *sql.DB) ([]Note, error) {
	rows, err := db.Query(`SELECT ` + noteColumns + ` FROM notes ORDER BY id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Note
	for rows.Next() {
		var n Note
		if err := rows.Scan(&n.ID, &n.ShortID, &n.Title, &n.Content, &n.CreatedAt, &n.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

func UpdateByShortID(db *sql.DB, shortID, title, content string) (*Note, error) {
	res, err := db.Exec(`UPDATE notes SET title = ?, content = ?, updated_at = datetime('now') WHERE short_id = ?`, title, content, shortID)
	if err != nil {
		return nil, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil, nil
	}
	return GetByShortID(db, shortID)
}

func DeleteByShortID(db *sql.DB, shortID string) (bool, error) {
	res, err := db.Exec(`DELETE FROM notes WHERE short_id = ?`, shortID)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}
