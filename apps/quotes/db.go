package main

import (
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/stevedylandev/andromeda/pkg/auth"
)

const quotesSchema = `
CREATE TABLE IF NOT EXISTS quotes (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    short_id   TEXT NOT NULL UNIQUE,
    text       TEXT NOT NULL,
    author     TEXT NOT NULL,
    source     TEXT,
    added_at   INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_quotes_added ON quotes(added_at DESC);
`

type Quote struct {
	ID        int64   `json:"id"`
	ShortID   string  `json:"short_id"`
	Text      string  `json:"text"`
	Author    string  `json:"author"`
	Source    *string `json:"source,omitempty"`
	AddedAt   int64   `json:"added_at"`
	UpdatedAt int64   `json:"updated_at"`
}

const selectCols = `id, short_id, text, author, source, added_at, updated_at`

func scanQuote(s interface{ Scan(...any) error }) (*Quote, error) {
	var q Quote
	var source sql.NullString
	err := s.Scan(&q.ID, &q.ShortID, &q.Text, &q.Author, &source, &q.AddedAt, &q.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if source.Valid {
		v := source.String
		q.Source = &v
	}
	return &q, nil
}

func countQuotes(db *sql.DB) (int, error) {
	var n int
	err := db.QueryRow(`SELECT COUNT(*) FROM quotes`).Scan(&n)
	return n, err
}

func listQuotes(db *sql.DB, limit int) ([]Quote, error) {
	rows, err := db.Query(`SELECT `+selectCols+` FROM quotes ORDER BY added_at DESC, id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Quote
	for rows.Next() {
		q, err := scanQuote(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *q)
	}
	return out, rows.Err()
}

func getQuoteByShortID(db *sql.DB, shortID string) (*Quote, error) {
	return scanQuote(db.QueryRow(`SELECT `+selectCols+` FROM quotes WHERE short_id = ?`, shortID))
}

// quoteOfTheDay returns a deterministic quote that is stable for the whole UTC
// day and rotates at midnight. Returns (nil, nil) when the table is empty.
func quoteOfTheDay(db *sql.DB) (*Quote, error) {
	n, err := countQuotes(db)
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, nil
	}
	offset := int(time.Now().UTC().Unix()/86400) % n
	return scanQuote(db.QueryRow(`SELECT `+selectCols+` FROM quotes ORDER BY id LIMIT 1 OFFSET ?`, offset))
}

func searchQuotes(db *sql.DB, q string) ([]Quote, error) {
	term := strings.TrimSpace(q)
	if term == "" {
		return nil, nil
	}
	pattern := "%" + strings.ToLower(term) + "%"
	rows, err := db.Query(
		`SELECT `+selectCols+` FROM quotes
		 WHERE LOWER(text) LIKE ? OR LOWER(author) LIKE ? OR LOWER(IFNULL(source,'')) LIKE ?
		 ORDER BY added_at DESC, id DESC LIMIT 50`,
		pattern, pattern, pattern,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Quote
	for rows.Next() {
		q, err := scanQuote(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *q)
	}
	return out, rows.Err()
}

// insertQuote inserts a quote with a freshly generated short id. source may be
// empty, in which case it is stored as NULL.
func insertQuote(db *sql.DB, text, author, source string) (int64, error) {
	shortID, err := auth.GenerateShortID(10)
	if err != nil {
		return 0, err
	}
	now := time.Now().UTC().Unix()
	var src any
	if strings.TrimSpace(source) != "" {
		src = source
	}
	res, err := db.Exec(
		`INSERT INTO quotes (short_id, text, author, source, added_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		shortID, text, author, src, now, now,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func deleteQuoteByShortID(db *sql.DB, shortID string) error {
	_, err := db.Exec(`DELETE FROM quotes WHERE short_id = ?`, shortID)
	return err
}
