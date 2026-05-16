package main

import (
	"database/sql"
	"strings"
)

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
