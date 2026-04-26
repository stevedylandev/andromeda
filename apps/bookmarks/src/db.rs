use andromeda_db::{Db, DbError};
use nanoid::nanoid;
use rusqlite::{OptionalExtension, params};
use serde::Serialize;

pub const SCHEMA: &str = r#"
CREATE TABLE IF NOT EXISTS categories (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    short_id   TEXT NOT NULL UNIQUE,
    name       TEXT NOT NULL UNIQUE,
    created_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS links (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    short_id    TEXT NOT NULL UNIQUE,
    title       TEXT NOT NULL,
    url         TEXT NOT NULL,
    category_id INTEGER NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    created_at  INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_links_category ON links(category_id, created_at DESC);
"#;

#[derive(Debug, Clone, Serialize)]
pub struct Category {
    pub id: i64,
    pub short_id: String,
    pub name: String,
}

#[derive(Debug, Clone, Serialize)]
pub struct Link {
    pub id: i64,
    pub short_id: String,
    pub title: String,
    pub url: String,
    pub category_id: i64,
    pub created_at: i64,
}

pub fn list_categories(db: &Db) -> Result<Vec<Category>, DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    let mut stmt = conn.prepare("SELECT id, short_id, name FROM categories ORDER BY name COLLATE NOCASE")?;
    let rows = stmt.query_map([], |row| {
        Ok(Category {
            id: row.get(0)?,
            short_id: row.get(1)?,
            name: row.get(2)?,
        })
    })?;
    Ok(rows.collect::<Result<Vec<_>, _>>()?)
}

pub fn create_category(db: &Db, name: &str) -> Result<Category, DbError> {
    let now = chrono::Utc::now().timestamp();
    let short_id = nanoid!(10);
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    conn.execute(
        "INSERT INTO categories (short_id, name, created_at) VALUES (?1, ?2, ?3)",
        params![short_id, name, now],
    )?;
    Ok(Category {
        id: conn.last_insert_rowid(),
        short_id,
        name: name.to_string(),
    })
}

pub fn delete_category_by_short_id(db: &Db, short_id: &str) -> Result<bool, DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    let n = conn.execute("DELETE FROM categories WHERE short_id = ?1", params![short_id])?;
    Ok(n > 0)
}

pub fn get_category_by_name(db: &Db, name: &str) -> Result<Option<Category>, DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    let cat = conn
        .query_row(
            "SELECT id, short_id, name FROM categories WHERE name = ?1",
            params![name],
            |row| {
                Ok(Category {
                    id: row.get(0)?,
                    short_id: row.get(1)?,
                    name: row.get(2)?,
                })
            },
        )
        .optional()?;
    Ok(cat)
}

pub fn list_links(db: &Db) -> Result<Vec<Link>, DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    let mut stmt = conn.prepare(
        "SELECT id, short_id, title, url, category_id, created_at FROM links ORDER BY created_at DESC",
    )?;
    let rows = stmt.query_map([], |row| {
        Ok(Link {
            id: row.get(0)?,
            short_id: row.get(1)?,
            title: row.get(2)?,
            url: row.get(3)?,
            category_id: row.get(4)?,
            created_at: row.get(5)?,
        })
    })?;
    Ok(rows.collect::<Result<Vec<_>, _>>()?)
}

pub fn create_link(db: &Db, title: &str, url: &str, category_id: i64) -> Result<Link, DbError> {
    let now = chrono::Utc::now().timestamp();
    let short_id = nanoid!(10);
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    conn.execute(
        "INSERT INTO links (short_id, title, url, category_id, created_at) VALUES (?1, ?2, ?3, ?4, ?5)",
        params![short_id, title, url, category_id, now],
    )?;
    Ok(Link {
        id: conn.last_insert_rowid(),
        short_id,
        title: title.to_string(),
        url: url.to_string(),
        category_id,
        created_at: now,
    })
}

pub fn delete_link_by_short_id(db: &Db, short_id: &str) -> Result<bool, DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    let n = conn.execute("DELETE FROM links WHERE short_id = ?1", params![short_id])?;
    Ok(n > 0)
}
