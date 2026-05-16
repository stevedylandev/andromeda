package main

import (
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/stevedylandev/andromeda/crates-go/auth"
	_ "modernc.org/sqlite"
)

const schema = `
CREATE TABLE IF NOT EXISTS categories (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    short_id   TEXT NOT NULL UNIQUE,
    name       TEXT NOT NULL UNIQUE,
    position   INTEGER NOT NULL DEFAULT 0,
    created_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS links (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    short_id     TEXT NOT NULL UNIQUE,
    title        TEXT NOT NULL,
    url          TEXT NOT NULL,
    favicon_url  TEXT,
    category_id  INTEGER NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    created_at   INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_links_category ON links(category_id, created_at DESC);
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

func listCategories(db *sql.DB) ([]Category, error) {
	rows, err := db.Query(`SELECT id, short_id, name, position FROM categories ORDER BY position ASC, name COLLATE NOCASE`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Category{}
	for rows.Next() {
		var c Category
		if err := rows.Scan(&c.ID, &c.ShortID, &c.Name, &c.Position); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func createCategory(db *sql.DB, name string) (*Category, error) {
	shortID, err := auth.GenerateShortID(10)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC().Unix()
	var nextPos int64
	if err := db.QueryRow(`SELECT COALESCE(MAX(position), 0) + 1 FROM categories`).Scan(&nextPos); err != nil {
		return nil, err
	}
	res, err := db.Exec(`INSERT INTO categories (short_id, name, position, created_at) VALUES (?, ?, ?, ?)`, shortID, name, nextPos, now)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return &Category{ID: id, ShortID: shortID, Name: name, Position: nextPos}, nil
}

func deleteCategoryByShortID(db *sql.DB, shortID string) (bool, error) {
	res, err := db.Exec(`DELETE FROM categories WHERE short_id = ?`, shortID)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func moveCategory(db *sql.DB, shortID string, direction int) (bool, error) {
	tx, err := db.Begin()
	if err != nil {
		return false, err
	}
	defer tx.Rollback()

	var curID, curPos int64
	err = tx.QueryRow(`SELECT id, position FROM categories WHERE short_id = ?`, shortID).Scan(&curID, &curPos)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	var nbID, nbPos int64
	if direction < 0 {
		err = tx.QueryRow(`SELECT id, position FROM categories WHERE position < ? ORDER BY position DESC LIMIT 1`, curPos).Scan(&nbID, &nbPos)
	} else {
		err = tx.QueryRow(`SELECT id, position FROM categories WHERE position > ? ORDER BY position ASC LIMIT 1`, curPos).Scan(&nbID, &nbPos)
	}
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	if _, err := tx.Exec(`UPDATE categories SET position = ? WHERE id = ?`, nbPos, curID); err != nil {
		return false, err
	}
	if _, err := tx.Exec(`UPDATE categories SET position = ? WHERE id = ?`, curPos, nbID); err != nil {
		return false, err
	}
	if err := tx.Commit(); err != nil {
		return false, err
	}
	return true, nil
}

func getCategoryByName(db *sql.DB, name string) (*Category, error) {
	name = strings.TrimSpace(name)
	var c Category
	err := db.QueryRow(`SELECT id, short_id, name, position FROM categories WHERE name = ?`, name).Scan(&c.ID, &c.ShortID, &c.Name, &c.Position)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func listLinks(db *sql.DB) ([]Link, error) {
	rows, err := db.Query(`SELECT id, short_id, title, url, favicon_url, category_id, created_at FROM links ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Link{}
	for rows.Next() {
		var l Link
		var fav sql.NullString
		if err := rows.Scan(&l.ID, &l.ShortID, &l.Title, &l.URL, &fav, &l.CategoryID, &l.CreatedAt); err != nil {
			return nil, err
		}
		if fav.Valid && fav.String != "" {
			s := fav.String
			l.FaviconURL = &s
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

func createLink(db *sql.DB, title, url string, faviconURL *string, categoryID int64) (*Link, error) {
	shortID, err := auth.GenerateShortID(10)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC().Unix()
	var fav any
	if faviconURL != nil && *faviconURL != "" {
		fav = *faviconURL
	}
	res, err := db.Exec(`INSERT INTO links (short_id, title, url, favicon_url, category_id, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		shortID, title, url, fav, categoryID, now)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	link := &Link{ID: id, ShortID: shortID, Title: title, URL: url, CategoryID: categoryID, CreatedAt: now}
	if faviconURL != nil && *faviconURL != "" {
		s := *faviconURL
		link.FaviconURL = &s
	}
	return link, nil
}

func listLinksMissingFavicon(db *sql.DB) ([]struct {
	ID  int64
	URL string
}, error) {
	rows, err := db.Query(`SELECT id, url FROM links WHERE favicon_url IS NULL OR favicon_url = ''`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []struct {
		ID  int64
		URL string
	}
	for rows.Next() {
		var r struct {
			ID  int64
			URL string
		}
		if err := rows.Scan(&r.ID, &r.URL); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func updateLinkFavicon(db *sql.DB, id int64, favicon *string) error {
	var v any
	if favicon != nil && *favicon != "" {
		v = *favicon
	}
	_, err := db.Exec(`UPDATE links SET favicon_url = ? WHERE id = ?`, v, id)
	return err
}

func deleteLinkByShortID(db *sql.DB, shortID string) (bool, error) {
	res, err := db.Exec(`DELETE FROM links WHERE short_id = ?`, shortID)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}
