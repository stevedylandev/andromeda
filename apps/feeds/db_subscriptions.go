package main

import (
	"database/sql"
	"errors"
	"time"
)

const subscriptionSelectColumns = `id, feed_url, title, site_url, favicon_url, category_id, etag, last_modified, last_fetched_at, last_error, added_at`

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
