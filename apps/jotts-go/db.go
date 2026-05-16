package main

import (
	"database/sql"
	"errors"
	"time"

	_ "modernc.org/sqlite"
)

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

CREATE TABLE IF NOT EXISTS sessions (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    token      TEXT NOT NULL UNIQUE,
    expires_at TEXT NOT NULL
);
`

func openDB(path string) (*sql.DB, error) {
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

func createNote(db *sql.DB, title, content string) (*Note, error) {
	shortID, err := generateShortID(10)
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

func getNoteByShortID(db *sql.DB, shortID string) (*Note, error) {
	return scanNote(db.QueryRow(`SELECT `+noteColumns+` FROM notes WHERE short_id = ?`, shortID))
}

func listNotes(db *sql.DB) ([]Note, error) {
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

func updateNoteByShortID(db *sql.DB, shortID, title, content string) (*Note, error) {
	res, err := db.Exec(`UPDATE notes SET title = ?, content = ?, updated_at = datetime('now') WHERE short_id = ?`, title, content, shortID)
	if err != nil {
		return nil, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil, nil
	}
	return getNoteByShortID(db, shortID)
}

func deleteNoteByShortID(db *sql.DB, shortID string) (bool, error) {
	res, err := db.Exec(`DELETE FROM notes WHERE short_id = ?`, shortID)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func createSession(db *sql.DB, token string, expiresAt time.Time) error {
	_, err := db.Exec(`INSERT INTO sessions (token, expires_at) VALUES (?, ?)`, token, expiresAt.UTC().Format("2006-01-02 15:04:05"))
	return err
}

func isValidSession(db *sql.DB, token string) bool {
	var expires string
	err := db.QueryRow(`SELECT expires_at FROM sessions WHERE token = ?`, token).Scan(&expires)
	if err != nil {
		return false
	}
	t, err := time.ParseInLocation("2006-01-02 15:04:05", expires, time.UTC)
	return err == nil && t.After(time.Now().UTC())
}

func deleteSession(db *sql.DB, token string) {
	_, _ = db.Exec(`DELETE FROM sessions WHERE token = ?`, token)
}

func pruneExpiredSessions(db *sql.DB) {
	_, _ = db.Exec(`DELETE FROM sessions WHERE expires_at < datetime('now')`)
}
