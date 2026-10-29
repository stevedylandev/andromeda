package main

import (
	"database/sql"
	"errors"
	"strings"
	"time"
)

const booksSchema = `
CREATE TABLE IF NOT EXISTS books (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    google_id     TEXT UNIQUE,
		ol_key        TEXT,
    title         TEXT NOT NULL,
    authors       TEXT NOT NULL,
    isbn          TEXT,
    cover_url     TEXT,
    notes         TEXT,
    status        TEXT NOT NULL CHECK (status IN ('read','reading','want')),
    added_at      INTEGER NOT NULL,
    updated_at    INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_books_status_added ON books(status, added_at DESC);

CREATE TABLE IF NOT EXISTS quotes (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    book_id    INTEGER NOT NULL REFERENCES books(id) ON DELETE CASCADE,
    page       TEXT,
    quote      TEXT NOT NULL,
    public     INTEGER NOT NULL DEFAULT 1,
    added_at   INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_quotes_book ON quotes(book_id, page);

CREATE TABLE IF NOT EXISTS settings (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);
`

type Book struct {
	ID        int64   `json:"id"`
	GoogleID  *string `json:"google_id,omitempty"`
	OLKey     *string `json:"key,omitempty"`
	Title     string  `json:"title"`
	Authors   string  `json:"authors"`
	ISBN      *string `json:"isbn,omitempty"`
	CoverURL  *string `json:"cover_url,omitempty"`
	Notes     *string `json:"notes,omitempty"`
	Status    string  `json:"status"`
	AddedAt   int64   `json:"added_at"`
	UpdatedAt int64   `json:"updated_at"`
}

type NewBook struct {
	GoogleID *string
	OLKey    *string
	Title    string
	Authors  string
	ISBN     *string
	CoverURL *string
	Notes    *string
	Status   string
}

type CategoryLabels struct {
	Reading string
	Read    string
	Want    string
}

func defaultLabels() CategoryLabels {
	return CategoryLabels{Reading: "Reading", Read: "Read", Want: "Want to Read"}
}

const selectCols = `id, google_id, ol_key, title, authors, isbn, cover_url, notes, status, added_at, updated_at`

func scanBook(s interface{ Scan(...any) error }) (*Book, error) {
	var b Book
	var gid, olKey, isbn, cover, notes sql.NullString
	err := s.Scan(&b.ID, &gid, &olKey, &b.Title, &b.Authors, &isbn, &cover, &notes, &b.Status, &b.AddedAt, &b.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if gid.Valid {
		v := gid.String
		b.GoogleID = &v
	}
	if olKey.Valid {
		v := olKey.String
		b.OLKey = &v
	}
	if isbn.Valid {
		v := isbn.String
		b.ISBN = &v
	}
	if cover.Valid {
		v := cover.String
		b.CoverURL = &v
	}
	if notes.Valid {
		v := notes.String
		b.Notes = &v
	}
	return &b, nil
}

func listBooks(db *sql.DB, status string) ([]Book, error) {
	var rows *sql.Rows
	var err error
	if status != "" {
		rows, err = db.Query(`SELECT `+selectCols+` FROM books WHERE status = ? ORDER BY added_at DESC`, status)
	} else {
		rows, err = db.Query(`SELECT ` + selectCols + ` FROM books ORDER BY added_at DESC`)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Book
	for rows.Next() {
		b, err := scanBook(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *b)
	}
	return out, rows.Err()
}

func getBook(db *sql.DB, id int64) (*Book, error) {
	return scanBook(db.QueryRow(`SELECT `+selectCols+` FROM books WHERE id = ?`, id))
}

func insertBook(db *sql.DB, b NewBook) (int64, error) {
	now := time.Now().UTC().Unix()
	var gid, olKey, isbn, cover, notes any
	if b.GoogleID != nil && *b.GoogleID != "" {
		gid = *b.GoogleID
	}
	if b.OLKey != nil && *b.OLKey != "" {
		olKey = *b.OLKey
	}
	if b.ISBN != nil && *b.ISBN != "" {
		isbn = *b.ISBN
	}
	if b.CoverURL != nil && *b.CoverURL != "" {
		cover = *b.CoverURL
	}
	if b.Notes != nil && *b.Notes != "" {
		notes = *b.Notes
	}
	res, err := db.Exec(
		`INSERT INTO books (google_id, ol_key, title, authors, isbn, cover_url, notes, status, added_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(google_id) DO UPDATE SET status = excluded.status, updated_at = excluded.updated_at`,
		gid, olKey, b.Title, b.Authors, isbn, cover, notes, b.Status, now, now,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func updateBookStatus(db *sql.DB, id int64, status string) error {
	_, err := db.Exec(`UPDATE books SET status = ?, updated_at = ? WHERE id = ?`, status, time.Now().UTC().Unix(), id)
	return err
}

func updateBookNotes(db *sql.DB, id int64, notes *string) error {
	var v any
	if notes != nil && *notes != "" {
		v = *notes
	}
	_, err := db.Exec(`UPDATE books SET notes = ?, updated_at = ? WHERE id = ?`, v, time.Now().UTC().Unix(), id)
	return err
}

func updateBookDetails(db *sql.DB, id int64, title, authors, isbn, coverURL, notes string) error {
	optional := func(s string) any {
		if strings.TrimSpace(s) == "" {
			return nil
		}
		return s
	}
	_, err := db.Exec(
		`UPDATE books SET title = ?, authors = ?, isbn = ?, cover_url = ?, notes = ?, updated_at = ? WHERE id = ?`,
		title, authors, optional(isbn), optional(coverURL), optional(notes), time.Now().UTC().Unix(), id,
	)
	return err
}

func deleteBook(db *sql.DB, id int64) error {
	_, err := db.Exec(`DELETE FROM books WHERE id = ?`, id)
	return err
}

func searchBooks(db *sql.DB, q string) ([]Book, error) {
	term := strings.TrimSpace(q)
	if term == "" {
		return nil, nil
	}
	pattern := "%" + strings.ToLower(term) + "%"
	rows, err := db.Query(
		`SELECT `+selectCols+` FROM books
		 WHERE LOWER(title) LIKE ? OR LOWER(authors) LIKE ? OR LOWER(IFNULL(isbn,'')) LIKE ?
		 ORDER BY added_at DESC LIMIT 50`,
		pattern, pattern, pattern,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Book
	for rows.Next() {
		b, err := scanBook(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *b)
	}
	return out, rows.Err()
}

func getSetting(db *sql.DB, key string) (string, bool, error) {
	var v string
	err := db.QueryRow(`SELECT value FROM settings WHERE key = ?`, key).Scan(&v)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return v, true, nil
}

func setSetting(db *sql.DB, key, value string) error {
	_, err := db.Exec(`INSERT INTO settings (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value`, key, value)
	return err
}

func getCategoryLabels(db *sql.DB) (CategoryLabels, error) {
	labels := defaultLabels()
	if v, ok, err := getSetting(db, "category_label.reading"); err == nil && ok {
		labels.Reading = v
	} else if err != nil {
		return labels, err
	}
	if v, ok, err := getSetting(db, "category_label.read"); err == nil && ok {
		labels.Read = v
	}
	if v, ok, err := getSetting(db, "category_label.want"); err == nil && ok {
		labels.Want = v
	}
	return labels, nil
}

func labelFor(l CategoryLabels, status string) string {
	switch status {
	case "reading":
		return l.Reading
	case "read":
		return l.Read
	case "want":
		return l.Want
	}
	return status
}

func validStatus(s string) bool {
	return s == "read" || s == "reading" || s == "want"
}

const quoteCols = `id, book_id, page, quote, public, added_at, updated_at`

type Quote struct {
	ID        int64   `json:"id"`
	BookID    int64   `json:"book_id"`
	Page      *string `json:"page,omitempty"`
	Quote     string  `json:"quote"`
	Public    bool    `json:"public"`
	AddedAt   int64   `json:"added_at"`
	UpdatedAt int64   `json:"updated_at"`
}

type NewQuote struct {
	BookID int64
	Page   *string
	Quote  string
	Public *bool // nil → true
}

type QuoteWithBook struct {
	ID          int64   `json:"id"`
	BookID      int64   `json:"book_id"`
	Page        *string `json:"page,omitempty"`
	Quote       string  `json:"quote"`
	Public      bool    `json:"public"`
	AddedAt     int64   `json:"added_at"`
	UpdatedAt   int64   `json:"updated_at"`
	BookTitle   string  `json:"book_title,omitempty"`
	BookAuthors string  `json:"book_authors,omitempty"`
}

func scanQuote(s interface{ Scan(...any) error }) (*Quote, error) {
	var q Quote
	var page sql.NullString
	var public bool
	err := s.Scan(&q.ID, &q.BookID, &page, &q.Quote, &public, &q.AddedAt, &q.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	q.Public = public
	if page.Valid {
		v := page.String
		q.Page = &v
	}
	return &q, nil
}

func insertQuote(db *sql.DB, q NewQuote) (int64, error) {
	now := time.Now().UTC().Unix()
	var page any
	if q.Page != nil && *q.Page != "" {
		page = *q.Page
	}
	public := true
	if q.Public != nil {
		public = *q.Public
	}
	res, err := db.Exec(
		`INSERT INTO quotes (book_id, page, quote, public, added_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		q.BookID, page, q.Quote, public, now, now,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func updateQuote(db *sql.DB, id int64, q NewQuote) error {
	var page any
	if q.Page != nil && *q.Page != "" {
		page = *q.Page
	}
	public := true
	if q.Public != nil {
		public = *q.Public
	}
	_, err := db.Exec(
		`UPDATE quotes SET book_id = ?, page = ?, quote = ?, public = ?, updated_at = ? WHERE id = ?`,
		q.BookID, page, q.Quote, public, time.Now().UTC().Unix(), id,
	)
	return err
}

func deleteQuote(db *sql.DB, id int64) error {
	_, err := db.Exec(`DELETE FROM quotes WHERE id = ?`, id)
	return err
}

func getQuote(db *sql.DB, id int64) (*Quote, error) {
	return scanQuote(db.QueryRow(`SELECT `+quoteCols+` FROM quotes WHERE id = ?`, id))
}

func listQuotesForBook(db *sql.DB, bookID int64, publicOnly bool) ([]Quote, error) {
	q := `SELECT ` + quoteCols + ` FROM quotes WHERE book_id = ?`
	if publicOnly {
		q += ` AND public = 1`
	}
	q += ` ORDER BY page ASC, added_at ASC`
	rows, err := db.Query(q, bookID)
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

func listQuotes(db *sql.DB, publicOnly bool) ([]QuoteWithBook, error) {
	q := `SELECT q.id, q.book_id, q.page, q.quote, q.public, q.added_at, q.updated_at, b.title, b.authors
	      FROM quotes q JOIN books b ON b.id = q.book_id`
	if publicOnly {
		q += ` WHERE q.public = 1`
	}
	q += ` ORDER BY b.title COLLATE NOCASE ASC, q.page ASC, q.added_at ASC`
	rows, err := db.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []QuoteWithBook
	for rows.Next() {
		var qb QuoteWithBook
		var page sql.NullString
		var public bool
		err := rows.Scan(&qb.ID, &qb.BookID, &page, &qb.Quote, &public, &qb.AddedAt, &qb.UpdatedAt, &qb.BookTitle, &qb.BookAuthors)
		if err != nil {
			return nil, err
		}
		qb.Public = public
		if page.Valid {
			v := page.String
			qb.Page = &v
		}
		out = append(out, qb)
	}
	return out, rows.Err()
}

func getQuoteWithBook(db *sql.DB, id int64) (*QuoteWithBook, error) {
	var qb QuoteWithBook
	var page sql.NullString
	var public bool
	err := db.QueryRow(
		`SELECT q.id, q.book_id, q.page, q.quote, q.public, q.added_at, q.updated_at, b.title, b.authors
		 FROM quotes q JOIN books b ON b.id = q.book_id WHERE q.id = ?`, id,
	).Scan(&qb.ID, &qb.BookID, &page, &qb.Quote, &public, &qb.AddedAt, &qb.UpdatedAt, &qb.BookTitle, &qb.BookAuthors)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	qb.Public = public
	if page.Valid {
		v := page.String
		qb.Page = &v
	}
	return &qb, nil
}
