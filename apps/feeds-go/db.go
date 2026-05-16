package main

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

const subscriptionSelectColumns = `id, feed_url, title, site_url, favicon_url, category_id, etag, last_modified, last_fetched_at, last_error, added_at`

const feedsSchema = `
CREATE TABLE IF NOT EXISTS categories (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    name       TEXT NOT NULL UNIQUE,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS subscriptions (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    feed_url        TEXT NOT NULL UNIQUE,
    title           TEXT NOT NULL,
    site_url        TEXT,
    favicon_url     TEXT,
    category_id     INTEGER REFERENCES categories(id) ON DELETE SET NULL,
    etag            TEXT,
    last_modified   TEXT,
    last_fetched_at TEXT,
    last_error      TEXT,
    added_at        TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_subs_category ON subscriptions(category_id);

CREATE TABLE IF NOT EXISTS items (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    subscription_id INTEGER NOT NULL REFERENCES subscriptions(id) ON DELETE CASCADE,
    guid            TEXT NOT NULL,
    title           TEXT NOT NULL,
    link            TEXT NOT NULL,
    author          TEXT,
    published_at    INTEGER NOT NULL,
    is_read         INTEGER NOT NULL DEFAULT 0,
    fetched_at      TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE(subscription_id, guid)
);
CREATE INDEX IF NOT EXISTS idx_items_sub_pub ON items(subscription_id, published_at DESC);
CREATE INDEX IF NOT EXISTS idx_items_pub ON items(published_at DESC);
CREATE INDEX IF NOT EXISTS idx_items_unread ON items(is_read, published_at DESC);

CREATE TABLE IF NOT EXISTS settings (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);
`

type Category struct {
	ID        int64
	Name      string
	CreatedAt string
}

type Subscription struct {
	ID            int64
	FeedURL       string
	Title         string
	SiteURL       sql.NullString
	FaviconURL    sql.NullString
	CategoryID    sql.NullInt64
	ETag          sql.NullString
	LastModified  sql.NullString
	LastFetchedAt sql.NullString
	LastError     sql.NullString
	AddedAt       string
}

type ItemWithFeed struct {
	ID             int64   `json:"id"`
	SubscriptionID int64   `json:"subscription_id"`
	GUID           string  `json:"guid"`
	Title          string  `json:"title"`
	Link           string  `json:"link"`
	Author         *string `json:"author,omitempty"`
	PublishedAt    int64   `json:"published_at"`
	IsRead         bool    `json:"is_read"`
	FetchedAt      string  `json:"fetched_at"`
	FeedTitle      string  `json:"feed_title"`
	FeedURL        string  `json:"feed_url"`
	CategoryID     *int64  `json:"category_id,omitempty"`
	CategoryName   *string `json:"category_name,omitempty"`
}

type ListItemsFilter struct {
	Limit          int
	UnreadOnly     bool
	CategoryID     *int64
	SubscriptionID *int64
}

type NewItem struct {
	SubscriptionID int64
	GUID           string
	Title          string
	Link           string
	Author         string
	PublishedAt    int64
}

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
	if _, err := db.Exec(feedsSchema); err != nil {
		return nil, err
	}
	return db, nil
}

func seedSettings(db *sql.DB, defaultPoll int) error {
	_, err := db.Exec(`INSERT INTO settings (key, value) VALUES ('poll_interval_minutes', ?)
		ON CONFLICT(key) DO NOTHING`, fmt.Sprintf("%d", defaultPoll))
	return err
}

func listItems(db *sql.DB, filter ListItemsFilter) ([]ItemWithFeed, error) {
	limit := filter.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}
	var b strings.Builder
	b.WriteString(`SELECT i.id, i.subscription_id, i.guid, i.title, i.link, i.author, i.published_at,
		i.is_read, i.fetched_at, s.title, s.feed_url, s.category_id, c.name
		FROM items i
		JOIN subscriptions s ON s.id = i.subscription_id
		LEFT JOIN categories c ON c.id = s.category_id
		WHERE 1=1`)
	args := []any{}
	if filter.UnreadOnly {
		b.WriteString(" AND i.is_read = 0")
	}
	if filter.CategoryID != nil {
		b.WriteString(" AND s.category_id = ?")
		args = append(args, *filter.CategoryID)
	}
	if filter.SubscriptionID != nil {
		b.WriteString(" AND i.subscription_id = ?")
		args = append(args, *filter.SubscriptionID)
	}
	b.WriteString(" ORDER BY i.published_at DESC, i.id DESC LIMIT ?")
	args = append(args, limit)
	rows, err := db.Query(b.String(), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []ItemWithFeed
	for rows.Next() {
		var it ItemWithFeed
		var author sql.NullString
		var categoryID sql.NullInt64
		var categoryName sql.NullString
		var isRead int
		if err := rows.Scan(&it.ID, &it.SubscriptionID, &it.GUID, &it.Title, &it.Link, &author, &it.PublishedAt, &isRead, &it.FetchedAt, &it.FeedTitle, &it.FeedURL, &categoryID, &categoryName); err != nil {
			return nil, err
		}
		if author.Valid {
			it.Author = &author.String
		}
		if categoryID.Valid {
			v := categoryID.Int64
			it.CategoryID = &v
		}
		if categoryName.Valid {
			v := categoryName.String
			it.CategoryName = &v
		}
		it.IsRead = isRead != 0
		items = append(items, it)
	}
	return items, rows.Err()
}

func listSubscriptions(db *sql.DB) ([]Subscription, error) {
	rows, err := db.Query(`SELECT ` + subscriptionSelectColumns + `
		FROM subscriptions ORDER BY title COLLATE NOCASE ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var subs []Subscription
	for rows.Next() {
		s, err := scanSubscription(rows)
		if err != nil {
			return nil, err
		}
		subs = append(subs, *s)
	}
	return subs, rows.Err()
}

func getSubscriptionByURL(db *sql.DB, feedURL string) (*Subscription, error) {
	return querySubscription(db, `SELECT `+subscriptionSelectColumns+` FROM subscriptions WHERE feed_url = ?`, feedURL)
}

func insertSubscription(db *sql.DB, feedURL, title string, siteURL *string, categoryID *int64) (*Subscription, error) {
	res, err := db.Exec(`INSERT INTO subscriptions (feed_url, title, site_url, category_id) VALUES (?, ?, ?, ?)`, feedURL, title, siteURL, categoryID)
	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	return getSubscription(db, id)
}

func getSubscription(db *sql.DB, id int64) (*Subscription, error) {
	return querySubscription(db, `SELECT `+subscriptionSelectColumns+` FROM subscriptions WHERE id = ?`, id)
}

func updateSubscriptionMeta(db *sql.DB, id int64, etag, lastModified *string, lastError *string) error {
	_, err := db.Exec(`UPDATE subscriptions SET etag = ?, last_modified = ?, last_fetched_at = ?, last_error = ? WHERE id = ?`,
		nullableString(etag), nullableString(lastModified), time.Now().UTC().Format("2006-01-02 15:04:05"), nullableString(lastError), id)
	return err
}

func updateSubscriptionTitle(db *sql.DB, id int64, title string) error {
	_, err := db.Exec(`UPDATE subscriptions SET title = ? WHERE id = ?`, title, id)
	return err
}

func updateSubscriptionSiteURL(db *sql.DB, id int64, siteURL *string) error {
	_, err := db.Exec(`UPDATE subscriptions SET site_url = ? WHERE id = ?`, nullableString(siteURL), id)
	return err
}

func updateSubscriptionFavicon(db *sql.DB, id int64, favicon *string) error {
	_, err := db.Exec(`UPDATE subscriptions SET favicon_url = ? WHERE id = ?`, nullableString(favicon), id)
	return err
}

func updateSubscriptionCategory(db *sql.DB, id int64, categoryID *int64) error {
	_, err := db.Exec(`UPDATE subscriptions SET category_id = ? WHERE id = ?`, nullableInt64(categoryID), id)
	return err
}

func deleteSubscription(db *sql.DB, id int64) (bool, error) {
	res, err := db.Exec(`DELETE FROM subscriptions WHERE id = ?`, id)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func insertItemIgnoreDup(db *sql.DB, item NewItem) (bool, error) {
	res, err := db.Exec(`INSERT OR IGNORE INTO items (subscription_id, guid, title, link, author, published_at) VALUES (?, ?, ?, ?, ?, ?)`,
		item.SubscriptionID, item.GUID, item.Title, item.Link, nullableString(stringPtr(strings.TrimSpace(item.Author))), item.PublishedAt)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func pruneSubscription(db *sql.DB, subscriptionID int64, keepN int) error {
	_, err := db.Exec(`DELETE FROM items
		WHERE subscription_id = ?
		  AND id NOT IN (
			SELECT id FROM items WHERE subscription_id = ? ORDER BY published_at DESC, id DESC LIMIT ?
		)`, subscriptionID, subscriptionID, keepN)
	return err
}

func markItemRead(db *sql.DB, id int64, isRead bool) (bool, error) {
	val := 0
	if isRead {
		val = 1
	}
	res, err := db.Exec(`UPDATE items SET is_read = ? WHERE id = ?`, val, id)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func listCategories(db *sql.DB) ([]Category, error) {
	rows, err := db.Query(`SELECT id, name, created_at FROM categories ORDER BY name ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Category
	for rows.Next() {
		var c Category
		if err := rows.Scan(&c.ID, &c.Name, &c.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func getOrCreateCategory(db *sql.DB, name string) (*Category, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, nil
	}
	var c Category
	err := db.QueryRow(`SELECT id, name, created_at FROM categories WHERE name = ?`, name).Scan(&c.ID, &c.Name, &c.CreatedAt)
	if err == nil {
		return &c, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	res, err := db.Exec(`INSERT INTO categories (name) VALUES (?)`, name)
	if err != nil {
		var existing Category
		if err2 := db.QueryRow(`SELECT id, name, created_at FROM categories WHERE name = ?`, name).Scan(&existing.ID, &existing.Name, &existing.CreatedAt); err2 == nil {
			return &existing, nil
		}
		return nil, err
	}
	id, _ := res.LastInsertId()
	return getCategory(db, id)
}

func getCategory(db *sql.DB, id int64) (*Category, error) {
	var c Category
	err := db.QueryRow(`SELECT id, name, created_at FROM categories WHERE id = ?`, id).Scan(&c.ID, &c.Name, &c.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func deleteCategory(db *sql.DB, id int64) (bool, error) {
	res, err := db.Exec(`DELETE FROM categories WHERE id = ?`, id)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func getSetting(db *sql.DB, key string) (string, bool, error) {
	var value string
	err := db.QueryRow(`SELECT value FROM settings WHERE key = ?`, key).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return value, true, nil
}

func setSetting(db *sql.DB, key, value string) error {
	_, err := db.Exec(`INSERT INTO settings (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value`, key, value)
	return err
}

func querySubscription(db *sql.DB, query string, args ...any) (*Subscription, error) {
	return scanSubscription(db.QueryRow(query, args...))
}

func scanSubscription(scanner interface{ Scan(dest ...any) error }) (*Subscription, error) {
	var s Subscription
	err := scanner.Scan(&s.ID, &s.FeedURL, &s.Title, &s.SiteURL, &s.FaviconURL, &s.CategoryID, &s.ETag, &s.LastModified, &s.LastFetchedAt, &s.LastError, &s.AddedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func nullableString(s *string) any {
	if s == nil || strings.TrimSpace(*s) == "" {
		return nil
	}
	return *s
}

func nullableInt64(v *int64) any {
	if v == nil {
		return nil
	}
	return *v
}

func stringPtr(s string) *string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return &s
}
