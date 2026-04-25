use andromeda_db::{Db, DbError};
use rusqlite::{params, OptionalExtension};
use serde::{Deserialize, Serialize};

pub const BOOKS_SCHEMA: &str = r#"
CREATE TABLE IF NOT EXISTS books (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    google_id     TEXT UNIQUE,
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
"#;

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum BookStatus {
    Read,
    Reading,
    Want,
}

impl BookStatus {
    pub fn as_str(&self) -> &'static str {
        match self {
            BookStatus::Read => "read",
            BookStatus::Reading => "reading",
            BookStatus::Want => "want",
        }
    }

    pub fn parse(s: &str) -> Option<Self> {
        match s {
            "read" => Some(BookStatus::Read),
            "reading" => Some(BookStatus::Reading),
            "want" => Some(BookStatus::Want),
            _ => None,
        }
    }
}

#[derive(Debug, Clone, Serialize)]
pub struct Book {
    pub id: i64,
    pub google_id: Option<String>,
    pub title: String,
    pub authors: String,
    pub isbn: Option<String>,
    pub cover_url: Option<String>,
    pub notes: Option<String>,
    pub status: String,
    pub added_at: i64,
    pub updated_at: i64,
}

#[derive(Debug, Clone)]
pub struct NewBook {
    pub google_id: Option<String>,
    pub title: String,
    pub authors: String,
    pub isbn: Option<String>,
    pub cover_url: Option<String>,
    pub notes: Option<String>,
    pub status: BookStatus,
}

fn map_book(row: &rusqlite::Row) -> rusqlite::Result<Book> {
    Ok(Book {
        id: row.get(0)?,
        google_id: row.get(1)?,
        title: row.get(2)?,
        authors: row.get(3)?,
        isbn: row.get(4)?,
        cover_url: row.get(5)?,
        notes: row.get(6)?,
        status: row.get(7)?,
        added_at: row.get(8)?,
        updated_at: row.get(9)?,
    })
}

const SELECT_COLS: &str =
    "id, google_id, title, authors, isbn, cover_url, notes, status, added_at, updated_at";

pub fn list_books(db: &Db, status: Option<BookStatus>) -> Result<Vec<Book>, DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    let books = match status {
        Some(s) => {
            let sql = format!(
                "SELECT {SELECT_COLS} FROM books WHERE status = ?1 ORDER BY added_at DESC"
            );
            let mut stmt = conn.prepare(&sql)?;
            let rows = stmt.query_map([s.as_str()], map_book)?;
            rows.collect::<Result<Vec<_>, _>>()?
        }
        None => {
            let sql = format!("SELECT {SELECT_COLS} FROM books ORDER BY added_at DESC");
            let mut stmt = conn.prepare(&sql)?;
            let rows = stmt.query_map([], map_book)?;
            rows.collect::<Result<Vec<_>, _>>()?
        }
    };
    Ok(books)
}

pub fn get_book(db: &Db, id: i64) -> Result<Option<Book>, DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    let sql = format!("SELECT {SELECT_COLS} FROM books WHERE id = ?1");
    let book = conn
        .query_row(&sql, [id], map_book)
        .optional()?;
    Ok(book)
}

pub fn insert_book(db: &Db, b: &NewBook) -> Result<i64, DbError> {
    let now = chrono::Utc::now().timestamp();
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    conn.execute(
        "INSERT INTO books (google_id, title, authors, isbn, cover_url, notes, status, added_at, updated_at)
         VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7, ?8, ?8)
         ON CONFLICT(google_id) DO UPDATE SET status = excluded.status, updated_at = excluded.updated_at",
        params![
            b.google_id,
            b.title,
            b.authors,
            b.isbn,
            b.cover_url,
            b.notes,
            b.status.as_str(),
            now,
        ],
    )?;
    Ok(conn.last_insert_rowid())
}

pub fn update_book_status(db: &Db, id: i64, status: BookStatus) -> Result<bool, DbError> {
    let now = chrono::Utc::now().timestamp();
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    let n = conn.execute(
        "UPDATE books SET status = ?1, updated_at = ?2 WHERE id = ?3",
        params![status.as_str(), now, id],
    )?;
    Ok(n > 0)
}

pub fn update_book_notes(db: &Db, id: i64, notes: Option<&str>) -> Result<bool, DbError> {
    let now = chrono::Utc::now().timestamp();
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    let n = conn.execute(
        "UPDATE books SET notes = ?1, updated_at = ?2 WHERE id = ?3",
        params![notes, now, id],
    )?;
    Ok(n > 0)
}

pub fn delete_book(db: &Db, id: i64) -> Result<bool, DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    let n = conn.execute("DELETE FROM books WHERE id = ?1", [id])?;
    Ok(n > 0)
}

#[derive(Debug, Default, Clone, Copy, Serialize)]
pub struct StatusCounts {
    pub read: i64,
    pub reading: i64,
    pub want: i64,
}

pub fn count_by_status(db: &Db) -> Result<StatusCounts, DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    let mut stmt = conn.prepare("SELECT status, COUNT(*) FROM books GROUP BY status")?;
    let mut counts = StatusCounts::default();
    let rows = stmt.query_map([], |row| {
        let s: String = row.get(0)?;
        let n: i64 = row.get(1)?;
        Ok((s, n))
    })?;
    for r in rows {
        let (s, n) = r?;
        match s.as_str() {
            "read" => counts.read = n,
            "reading" => counts.reading = n,
            "want" => counts.want = n,
            _ => {}
        }
    }
    Ok(counts)
}
