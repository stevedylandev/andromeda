use rusqlite::{params, OptionalExtension, Row};
use serde::{Deserialize, Serialize};

use crate::{Db, DbError};

pub const FEEDS_SCHEMA: &str = "
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
";

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Category {
    pub id: i64,
    pub name: String,
    pub created_at: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Subscription {
    pub id: i64,
    pub feed_url: String,
    pub title: String,
    pub site_url: Option<String>,
    pub favicon_url: Option<String>,
    pub category_id: Option<i64>,
    pub etag: Option<String>,
    pub last_modified: Option<String>,
    pub last_fetched_at: Option<String>,
    pub last_error: Option<String>,
    pub added_at: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Item {
    pub id: i64,
    pub subscription_id: i64,
    pub guid: String,
    pub title: String,
    pub link: String,
    pub author: Option<String>,
    pub published_at: i64,
    pub is_read: bool,
    pub fetched_at: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ItemWithFeed {
    pub id: i64,
    pub subscription_id: i64,
    pub guid: String,
    pub title: String,
    pub link: String,
    pub author: Option<String>,
    pub published_at: i64,
    pub is_read: bool,
    pub fetched_at: String,
    pub feed_title: String,
    pub feed_url: String,
    pub category_id: Option<i64>,
    pub category_name: Option<String>,
}

#[derive(Debug, Clone)]
pub struct NewItem<'a> {
    pub subscription_id: i64,
    pub guid: &'a str,
    pub title: &'a str,
    pub link: &'a str,
    pub author: Option<&'a str>,
    pub published_at: i64,
}

fn category_from_row(row: &Row) -> rusqlite::Result<Category> {
    Ok(Category {
        id: row.get(0)?,
        name: row.get(1)?,
        created_at: row.get(2)?,
    })
}

fn subscription_from_row(row: &Row) -> rusqlite::Result<Subscription> {
    Ok(Subscription {
        id: row.get(0)?,
        feed_url: row.get(1)?,
        title: row.get(2)?,
        site_url: row.get(3)?,
        favicon_url: row.get(4)?,
        category_id: row.get(5)?,
        etag: row.get(6)?,
        last_modified: row.get(7)?,
        last_fetched_at: row.get(8)?,
        last_error: row.get(9)?,
        added_at: row.get(10)?,
    })
}

const SUB_COLS: &str = "id, feed_url, title, site_url, favicon_url, category_id, etag, last_modified, last_fetched_at, last_error, added_at";

/// Add columns introduced after the initial schema. Idempotent.
pub fn migrate_feeds(db: &Db) -> Result<(), DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    let has_favicon: bool = conn
        .query_row(
            "SELECT 1 FROM pragma_table_info('subscriptions') WHERE name = 'favicon_url'",
            [],
            |_| Ok(true),
        )
        .optional()?
        .unwrap_or(false);
    if !has_favicon {
        conn.execute("ALTER TABLE subscriptions ADD COLUMN favicon_url TEXT", [])?;
    }
    Ok(())
}

pub fn update_subscription_favicon(
    db: &Db,
    id: i64,
    favicon_url: Option<&str>,
) -> Result<(), DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    conn.execute(
        "UPDATE subscriptions SET favicon_url = ?1 WHERE id = ?2",
        params![favicon_url, id],
    )?;
    Ok(())
}

pub fn update_subscription_site_url(
    db: &Db,
    id: i64,
    site_url: Option<&str>,
) -> Result<(), DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    conn.execute(
        "UPDATE subscriptions SET site_url = ?1 WHERE id = ?2",
        params![site_url, id],
    )?;
    Ok(())
}

// ── Categories ────────────────────────────────────────────────────────

pub fn list_categories(db: &Db) -> Result<Vec<Category>, DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    let mut stmt = conn.prepare("SELECT id, name, created_at FROM categories ORDER BY name ASC")?;
    let rows = stmt
        .query_map([], category_from_row)?
        .collect::<Result<Vec<_>, _>>()?;
    Ok(rows)
}

pub fn insert_category(db: &Db, name: &str) -> Result<Category, DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    conn.execute("INSERT INTO categories (name) VALUES (?1)", params![name])?;
    let id = conn.last_insert_rowid();
    let cat = conn.query_row(
        "SELECT id, name, created_at FROM categories WHERE id = ?1",
        params![id],
        category_from_row,
    )?;
    Ok(cat)
}

pub fn get_or_create_category(db: &Db, name: &str) -> Result<Category, DbError> {
    {
        let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
        if let Some(cat) = conn
            .query_row(
                "SELECT id, name, created_at FROM categories WHERE name = ?1",
                params![name],
                category_from_row,
            )
            .optional()?
        {
            return Ok(cat);
        }
    }
    insert_category(db, name)
}

pub fn delete_category(db: &Db, id: i64) -> Result<bool, DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    let rows = conn.execute("DELETE FROM categories WHERE id = ?1", params![id])?;
    Ok(rows > 0)
}

// ── Subscriptions ─────────────────────────────────────────────────────

pub fn list_subscriptions(db: &Db) -> Result<Vec<Subscription>, DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    let mut stmt = conn.prepare(&format!(
        "SELECT {SUB_COLS} FROM subscriptions ORDER BY title COLLATE NOCASE ASC"
    ))?;
    let rows = stmt
        .query_map([], subscription_from_row)?
        .collect::<Result<Vec<_>, _>>()?;
    Ok(rows)
}

pub fn get_subscription(db: &Db, id: i64) -> Result<Option<Subscription>, DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    let sub = conn
        .query_row(
            &format!("SELECT {SUB_COLS} FROM subscriptions WHERE id = ?1"),
            params![id],
            subscription_from_row,
        )
        .optional()?;
    Ok(sub)
}

pub fn get_subscription_by_url(db: &Db, feed_url: &str) -> Result<Option<Subscription>, DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    let sub = conn
        .query_row(
            &format!("SELECT {SUB_COLS} FROM subscriptions WHERE feed_url = ?1"),
            params![feed_url],
            subscription_from_row,
        )
        .optional()?;
    Ok(sub)
}

pub fn insert_subscription(
    db: &Db,
    feed_url: &str,
    title: &str,
    site_url: Option<&str>,
    category_id: Option<i64>,
) -> Result<Subscription, DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    conn.execute(
        "INSERT INTO subscriptions (feed_url, title, site_url, category_id) VALUES (?1, ?2, ?3, ?4)",
        params![feed_url, title, site_url, category_id],
    )?;
    let id = conn.last_insert_rowid();
    let sub = conn.query_row(
        &format!("SELECT {SUB_COLS} FROM subscriptions WHERE id = ?1"),
        params![id],
        subscription_from_row,
    )?;
    Ok(sub)
}

pub fn update_subscription_meta(
    db: &Db,
    id: i64,
    etag: Option<&str>,
    last_modified: Option<&str>,
    last_fetched_at: &str,
    last_error: Option<&str>,
) -> Result<(), DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    conn.execute(
        "UPDATE subscriptions SET etag = ?1, last_modified = ?2, last_fetched_at = ?3, last_error = ?4 WHERE id = ?5",
        params![etag, last_modified, last_fetched_at, last_error, id],
    )?;
    Ok(())
}

pub fn update_subscription_title(db: &Db, id: i64, title: &str) -> Result<(), DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    conn.execute(
        "UPDATE subscriptions SET title = ?1 WHERE id = ?2",
        params![title, id],
    )?;
    Ok(())
}

pub fn update_subscription_category(
    db: &Db,
    id: i64,
    category_id: Option<i64>,
) -> Result<(), DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    conn.execute(
        "UPDATE subscriptions SET category_id = ?1 WHERE id = ?2",
        params![category_id, id],
    )?;
    Ok(())
}

pub fn delete_subscription(db: &Db, id: i64) -> Result<bool, DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    let rows = conn.execute("DELETE FROM subscriptions WHERE id = ?1", params![id])?;
    Ok(rows > 0)
}

// ── Items ─────────────────────────────────────────────────────────────

/// Insert if new. Returns true if inserted, false if a duplicate (guid) existed.
pub fn insert_item_ignore_dup(db: &Db, item: &NewItem<'_>) -> Result<bool, DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    let rows = conn.execute(
        "INSERT OR IGNORE INTO items (subscription_id, guid, title, link, author, published_at)
         VALUES (?1, ?2, ?3, ?4, ?5, ?6)",
        params![
            item.subscription_id,
            item.guid,
            item.title,
            item.link,
            item.author,
            item.published_at,
        ],
    )?;
    Ok(rows > 0)
}

#[derive(Debug, Clone, Default)]
pub struct ListItemsFilter {
    pub limit: Option<i64>,
    pub unread_only: bool,
    pub category_id: Option<i64>,
    pub subscription_id: Option<i64>,
}

pub fn list_items(db: &Db, filter: &ListItemsFilter) -> Result<Vec<ItemWithFeed>, DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;

    let mut sql = String::from(
        "SELECT i.id, i.subscription_id, i.guid, i.title, i.link, i.author, i.published_at,
                i.is_read, i.fetched_at, s.title, s.feed_url, s.category_id, c.name
         FROM items i
         JOIN subscriptions s ON s.id = i.subscription_id
         LEFT JOIN categories c ON c.id = s.category_id
         WHERE 1=1",
    );
    let mut binds: Vec<Box<dyn rusqlite::ToSql>> = Vec::new();

    if filter.unread_only {
        sql.push_str(" AND i.is_read = 0");
    }
    if let Some(cid) = filter.category_id {
        sql.push_str(&format!(" AND s.category_id = ?{}", binds.len() + 1));
        binds.push(Box::new(cid));
    }
    if let Some(sid) = filter.subscription_id {
        sql.push_str(&format!(" AND i.subscription_id = ?{}", binds.len() + 1));
        binds.push(Box::new(sid));
    }

    sql.push_str(" ORDER BY i.published_at DESC, i.id DESC");

    let limit = filter.limit.unwrap_or(100).clamp(1, 1000);
    sql.push_str(&format!(" LIMIT ?{}", binds.len() + 1));
    binds.push(Box::new(limit));

    let mut stmt = conn.prepare(&sql)?;
    let params_slice: Vec<&dyn rusqlite::ToSql> = binds.iter().map(|b| b.as_ref()).collect();
    let rows = stmt
        .query_map(params_slice.as_slice(), |row| {
            Ok(ItemWithFeed {
                id: row.get(0)?,
                subscription_id: row.get(1)?,
                guid: row.get(2)?,
                title: row.get(3)?,
                link: row.get(4)?,
                author: row.get(5)?,
                published_at: row.get(6)?,
                is_read: row.get::<_, i64>(7)? != 0,
                fetched_at: row.get(8)?,
                feed_title: row.get(9)?,
                feed_url: row.get(10)?,
                category_id: row.get(11)?,
                category_name: row.get(12)?,
            })
        })?
        .collect::<Result<Vec<_>, _>>()?;
    Ok(rows)
}

pub fn mark_read(db: &Db, id: i64) -> Result<bool, DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    let rows = conn.execute("UPDATE items SET is_read = 1 WHERE id = ?1", params![id])?;
    Ok(rows > 0)
}

pub fn mark_unread(db: &Db, id: i64) -> Result<bool, DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    let rows = conn.execute("UPDATE items SET is_read = 0 WHERE id = ?1", params![id])?;
    Ok(rows > 0)
}

/// Keep newest `keep_n` items for a subscription, delete older.
pub fn prune_subscription(db: &Db, subscription_id: i64, keep_n: i64) -> Result<usize, DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    let rows = conn.execute(
        "DELETE FROM items
         WHERE subscription_id = ?1
           AND id NOT IN (
             SELECT id FROM items
             WHERE subscription_id = ?1
             ORDER BY published_at DESC, id DESC
             LIMIT ?2
           )",
        params![subscription_id, keep_n],
    )?;
    Ok(rows)
}

// ── Settings ──────────────────────────────────────────────────────────

pub fn get_setting(db: &Db, key: &str) -> Result<Option<String>, DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    let val = conn
        .query_row(
            "SELECT value FROM settings WHERE key = ?1",
            params![key],
            |row| row.get::<_, String>(0),
        )
        .optional()?;
    Ok(val)
}

pub fn set_setting(db: &Db, key: &str, value: &str) -> Result<(), DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    conn.execute(
        "INSERT INTO settings (key, value) VALUES (?1, ?2)
         ON CONFLICT(key) DO UPDATE SET value = excluded.value",
        params![key, value],
    )?;
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    use rusqlite::Connection;
    use std::sync::{Arc, Mutex};

    fn test_db() -> Db {
        let conn = Connection::open_in_memory().unwrap();
        conn.execute_batch(FEEDS_SCHEMA).unwrap();
        Arc::new(Mutex::new(conn))
    }

    #[test]
    fn category_crud() {
        let db = test_db();
        let cat = insert_category(&db, "Tech").unwrap();
        assert_eq!(cat.name, "Tech");

        let all = list_categories(&db).unwrap();
        assert_eq!(all.len(), 1);

        let same = get_or_create_category(&db, "Tech").unwrap();
        assert_eq!(same.id, cat.id);

        let other = get_or_create_category(&db, "News").unwrap();
        assert_ne!(other.id, cat.id);

        assert!(delete_category(&db, cat.id).unwrap());
        assert_eq!(list_categories(&db).unwrap().len(), 1);
    }

    #[test]
    fn subscription_crud() {
        let db = test_db();
        let sub = insert_subscription(
            &db,
            "https://example.com/feed",
            "Example",
            Some("https://example.com"),
            None,
        )
        .unwrap();
        assert_eq!(sub.title, "Example");

        let fetched = get_subscription_by_url(&db, "https://example.com/feed")
            .unwrap()
            .unwrap();
        assert_eq!(fetched.id, sub.id);

        update_subscription_meta(
            &db,
            sub.id,
            Some("etag-1"),
            Some("Sun, 01 Jan 2024 00:00:00 GMT"),
            "2024-01-01 00:00:00",
            None,
        )
        .unwrap();
        let after = get_subscription(&db, sub.id).unwrap().unwrap();
        assert_eq!(after.etag.as_deref(), Some("etag-1"));

        assert!(delete_subscription(&db, sub.id).unwrap());
        assert!(get_subscription(&db, sub.id).unwrap().is_none());
    }

    #[test]
    fn item_insert_dedup_and_list() {
        let db = test_db();
        let sub = insert_subscription(&db, "https://a.com/feed", "A", None, None).unwrap();

        let inserted = insert_item_ignore_dup(
            &db,
            &NewItem {
                subscription_id: sub.id,
                guid: "g1",
                title: "Post 1",
                link: "https://a.com/1",
                author: Some("Alice"),
                published_at: 1_700_000_000,
            },
        )
        .unwrap();
        assert!(inserted);

        let dup = insert_item_ignore_dup(
            &db,
            &NewItem {
                subscription_id: sub.id,
                guid: "g1",
                title: "Post 1 different title",
                link: "https://a.com/1",
                author: None,
                published_at: 1_700_000_000,
            },
        )
        .unwrap();
        assert!(!dup);

        let items = list_items(&db, &ListItemsFilter::default()).unwrap();
        assert_eq!(items.len(), 1);
        assert_eq!(items[0].title, "Post 1");
        assert_eq!(items[0].feed_title, "A");
        assert!(!items[0].is_read);
    }

    #[test]
    fn mark_read_unread() {
        let db = test_db();
        let sub = insert_subscription(&db, "https://a.com/feed", "A", None, None).unwrap();
        insert_item_ignore_dup(
            &db,
            &NewItem {
                subscription_id: sub.id,
                guid: "g",
                title: "t",
                link: "l",
                author: None,
                published_at: 1,
            },
        )
        .unwrap();
        let items = list_items(&db, &ListItemsFilter::default()).unwrap();
        let id = items[0].id;

        assert!(mark_read(&db, id).unwrap());
        let read = list_items(
            &db,
            &ListItemsFilter {
                unread_only: true,
                ..Default::default()
            },
        )
        .unwrap();
        assert_eq!(read.len(), 0);

        assert!(mark_unread(&db, id).unwrap());
        let unread = list_items(
            &db,
            &ListItemsFilter {
                unread_only: true,
                ..Default::default()
            },
        )
        .unwrap();
        assert_eq!(unread.len(), 1);
    }

    #[test]
    fn prune_keeps_newest() {
        let db = test_db();
        let sub = insert_subscription(&db, "https://a.com/feed", "A", None, None).unwrap();
        for i in 0..10 {
            insert_item_ignore_dup(
                &db,
                &NewItem {
                    subscription_id: sub.id,
                    guid: &format!("g{i}"),
                    title: "t",
                    link: "l",
                    author: None,
                    published_at: i as i64,
                },
            )
            .unwrap();
        }
        let removed = prune_subscription(&db, sub.id, 3).unwrap();
        assert_eq!(removed, 7);

        let items = list_items(&db, &ListItemsFilter::default()).unwrap();
        assert_eq!(items.len(), 3);
        assert_eq!(items[0].published_at, 9);
        assert_eq!(items[2].published_at, 7);
    }

    #[test]
    fn settings_upsert() {
        let db = test_db();
        assert!(get_setting(&db, "poll").unwrap().is_none());
        set_setting(&db, "poll", "30").unwrap();
        assert_eq!(get_setting(&db, "poll").unwrap().as_deref(), Some("30"));
        set_setting(&db, "poll", "60").unwrap();
        assert_eq!(get_setting(&db, "poll").unwrap().as_deref(), Some("60"));
    }

    #[test]
    fn category_filter_on_items() {
        let db = test_db();
        let tech = insert_category(&db, "Tech").unwrap();
        let sub_tech =
            insert_subscription(&db, "https://a.com/feed", "A", None, Some(tech.id)).unwrap();
        let sub_other = insert_subscription(&db, "https://b.com/feed", "B", None, None).unwrap();

        insert_item_ignore_dup(
            &db,
            &NewItem {
                subscription_id: sub_tech.id,
                guid: "g1",
                title: "tech post",
                link: "",
                author: None,
                published_at: 1,
            },
        )
        .unwrap();
        insert_item_ignore_dup(
            &db,
            &NewItem {
                subscription_id: sub_other.id,
                guid: "g2",
                title: "other post",
                link: "",
                author: None,
                published_at: 2,
            },
        )
        .unwrap();

        let tech_items = list_items(
            &db,
            &ListItemsFilter {
                category_id: Some(tech.id),
                ..Default::default()
            },
        )
        .unwrap();
        assert_eq!(tech_items.len(), 1);
        assert_eq!(tech_items[0].title, "tech post");
        assert_eq!(tech_items[0].category_name.as_deref(), Some("Tech"));
    }
}
