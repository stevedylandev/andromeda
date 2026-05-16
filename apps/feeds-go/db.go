package main

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

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

func seedSettings(db *sql.DB, defaultPoll int) error {
	_, err := db.Exec(`INSERT INTO settings (key, value) VALUES ('poll_interval_minutes', ?)
		ON CONFLICT(key) DO NOTHING`, fmt.Sprintf("%d", defaultPoll))
	return err
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
