package store

import (
	"database/sql"
	"errors"

	"github.com/stevedylandev/andromeda/crates-go/auth"
	_ "modernc.org/sqlite"
)

type Snippet struct {
	ID      int64  `json:"id"`
	ShortID string `json:"short_id"`
	Content string `json:"content"`
	Name    string `json:"name"`
}

type SnippetInput struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

const schema = `
CREATE TABLE IF NOT EXISTS snippets (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    short_id TEXT NOT NULL UNIQUE,
    content TEXT NOT NULL,
    name TEXT NOT NULL
);
`

func Open(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if _, err := db.Exec(schema); err != nil {
		return nil, err
	}
	return db, nil
}

func scanSnippet(s interface{ Scan(...any) error }) (*Snippet, error) {
	var sn Snippet
	err := s.Scan(&sn.ID, &sn.ShortID, &sn.Content, &sn.Name)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &sn, nil
}

func Create(db *sql.DB, name, content string) (*Snippet, error) {
	shortID, err := auth.GenerateShortID(10)
	if err != nil {
		return nil, err
	}
	res, err := db.Exec(`INSERT INTO snippets (short_id, content, name) VALUES (?, ?, ?)`, shortID, content, name)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return &Snippet{ID: id, ShortID: shortID, Content: content, Name: name}, nil
}

func GetByShortID(db *sql.DB, shortID string) (*Snippet, error) {
	return scanSnippet(db.QueryRow(`SELECT id, short_id, content, name FROM snippets WHERE short_id = ?`, shortID))
}

func List(db *sql.DB) ([]Snippet, error) {
	rows, err := db.Query(`SELECT id, short_id, content, name FROM snippets ORDER BY id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Snippet{}
	for rows.Next() {
		s, err := scanSnippet(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *s)
	}
	return out, rows.Err()
}

func DeleteByShortID(db *sql.DB, shortID string) (bool, error) {
	res, err := db.Exec(`DELETE FROM snippets WHERE short_id = ?`, shortID)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func UpdateByShortID(db *sql.DB, shortID, name, content string) (*Snippet, error) {
	res, err := db.Exec(`UPDATE snippets SET name = ?, content = ? WHERE short_id = ?`, name, content, shortID)
	if err != nil {
		return nil, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil, nil
	}
	return GetByShortID(db, shortID)
}
