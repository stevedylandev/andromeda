package store

import (
	"database/sql"
	"errors"

	"github.com/stevedylandev/andromeda/pkg/auth"
	"github.com/stevedylandev/andromeda/pkg/sqlite"
)

type Note struct {
	ID        int64  `json:"id"`
	ShortID   string `json:"short_id"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
	Public    bool   `json:"public"`
}

type NoteInput struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

type UpdateNoteRequest struct {
    Title   string `json:"title"`
    Content string `json:"content"`
    Public  *bool  `json:"public,omitempty"`
}

const noteColumns = `id, short_id, title, content, created_at, updated_at, public`

const schema = `
CREATE TABLE IF NOT EXISTS notes (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    short_id   TEXT NOT NULL UNIQUE,
    title      TEXT NOT NULL,
    content    TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now')),
		public     BOOL          DEFAULT (false)
);
`

func Open(path string) (*sql.DB, error) {
	db, err := sqlite.Open(path, schema)
	if err != nil {
		return nil, err
	}
	db.Exec(`ALTER TABLE notes ADD COLUMN public BOOL DEFAULT (false)`)
	return db, nil
}

func scanNote(scanner interface{ Scan(dest ...any) error }) (*Note, error) {
	var n Note
	err := scanner.Scan(&n.ID, &n.ShortID, &n.Title, &n.Content, &n.CreatedAt, &n.UpdatedAt, &n.Public)
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
	rows, err := db.Query(`SELECT ` + noteColumns + ` FROM notes ORDER BY updated_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Note
	for rows.Next() {
		var n Note
		if err := rows.Scan(&n.ID, &n.ShortID, &n.Title, &n.Content, &n.CreatedAt, &n.UpdatedAt, &n.Public); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}


func UpdateByShortID(db *sql.DB, shortID, title, content string, public *bool) (*Note, error) {
    res, err := db.Exec(
        `UPDATE notes
         SET title = ?, content = ?, public = COALESCE(?, public), updated_at = datetime('now')
         WHERE short_id = ?`,
        title, content, public, shortID,
    )
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
