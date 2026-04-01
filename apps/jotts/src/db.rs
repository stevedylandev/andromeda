use nanoid::nanoid;
use rusqlite::{Connection, params};
use serde::{Deserialize, Serialize};
use std::fmt;
use std::sync::{Arc, Mutex};

pub type Db = Arc<Mutex<Connection>>;

#[derive(Debug)]
pub enum DbError {
    Sqlite(rusqlite::Error),
    LockPoisoned,
}

impl fmt::Display for DbError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            DbError::Sqlite(e) => write!(f, "Database error: {}", e),
            DbError::LockPoisoned => write!(f, "Database lock poisoned"),
        }
    }
}

impl std::error::Error for DbError {}

impl From<rusqlite::Error> for DbError {
    fn from(e: rusqlite::Error) -> Self {
        DbError::Sqlite(e)
    }
}

#[derive(Debug, Serialize, Deserialize)]
pub struct Note {
    pub id: i64,
    pub short_id: String,
    pub title: String,
    pub content: String,
    pub created_at: String,
    pub updated_at: String,
}

pub fn init_db() -> Db {
    let path = std::env::var("JOTTS_DB_PATH").unwrap_or_else(|_| "jotts.sqlite".to_string());
    let conn = Connection::open(&path).expect("Failed to open database");

    conn.execute_batch(
        "CREATE TABLE IF NOT EXISTS notes (
            id         INTEGER PRIMARY KEY AUTOINCREMENT,
            short_id   TEXT NOT NULL UNIQUE,
            title      TEXT NOT NULL,
            content    TEXT NOT NULL,
            created_at TEXT NOT NULL DEFAULT (datetime('now')),
            updated_at TEXT NOT NULL DEFAULT (datetime('now'))
        );

        CREATE TABLE IF NOT EXISTS sessions (
            id         INTEGER PRIMARY KEY AUTOINCREMENT,
            token      TEXT NOT NULL UNIQUE,
            expires_at TEXT NOT NULL
        );"
    )
    .expect("Failed to create tables");

    Arc::new(Mutex::new(conn))
}

pub fn create_note(db: &Db, title: &str, content: &str) -> Result<Note, DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    let short_id = nanoid!(10);
    conn.execute(
        "INSERT INTO notes (short_id, title, content) VALUES (?1, ?2, ?3)",
        params![short_id, title, content],
    )?;
    let id = conn.last_insert_rowid();
    let note = conn.query_row(
        "SELECT id, short_id, title, content, created_at, updated_at FROM notes WHERE id = ?1",
        params![id],
        |row| {
            Ok(Note {
                id: row.get(0)?,
                short_id: row.get(1)?,
                title: row.get(2)?,
                content: row.get(3)?,
                created_at: row.get(4)?,
                updated_at: row.get(5)?,
            })
        },
    )?;
    Ok(note)
}

pub fn get_note_by_short_id(db: &Db, short_id: &str) -> Result<Option<Note>, DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    match conn.query_row(
        "SELECT id, short_id, title, content, created_at, updated_at FROM notes WHERE short_id = ?1",
        params![short_id],
        |row| {
            Ok(Note {
                id: row.get(0)?,
                short_id: row.get(1)?,
                title: row.get(2)?,
                content: row.get(3)?,
                created_at: row.get(4)?,
                updated_at: row.get(5)?,
            })
        },
    ) {
        Ok(note) => Ok(Some(note)),
        Err(rusqlite::Error::QueryReturnedNoRows) => Ok(None),
        Err(e) => Err(DbError::Sqlite(e)),
    }
}

pub fn get_all_notes(db: &Db) -> Result<Vec<Note>, DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    let mut stmt = conn.prepare(
        "SELECT id, short_id, title, content, created_at, updated_at FROM notes ORDER BY id DESC",
    )?;
    let notes = stmt
        .query_map([], |row| {
            Ok(Note {
                id: row.get(0)?,
                short_id: row.get(1)?,
                title: row.get(2)?,
                content: row.get(3)?,
                created_at: row.get(4)?,
                updated_at: row.get(5)?,
            })
        })?
        .collect::<Result<Vec<_>, _>>()?;
    Ok(notes)
}

pub fn update_note_by_short_id(
    db: &Db,
    short_id: &str,
    title: &str,
    content: &str,
) -> Result<Option<Note>, DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    let rows = conn.execute(
        "UPDATE notes SET title = ?1, content = ?2, updated_at = datetime('now') WHERE short_id = ?3",
        params![title, content, short_id],
    )?;
    if rows == 0 {
        return Ok(None);
    }
    match conn.query_row(
        "SELECT id, short_id, title, content, created_at, updated_at FROM notes WHERE short_id = ?1",
        params![short_id],
        |row| {
            Ok(Note {
                id: row.get(0)?,
                short_id: row.get(1)?,
                title: row.get(2)?,
                content: row.get(3)?,
                created_at: row.get(4)?,
                updated_at: row.get(5)?,
            })
        },
    ) {
        Ok(note) => Ok(Some(note)),
        Err(rusqlite::Error::QueryReturnedNoRows) => Ok(None),
        Err(e) => Err(DbError::Sqlite(e)),
    }
}

pub fn delete_note_by_short_id(db: &Db, short_id: &str) -> Result<bool, DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    let rows = conn.execute(
        "DELETE FROM notes WHERE short_id = ?1",
        params![short_id],
    )?;
    Ok(rows > 0)
}

// Session functions

pub fn insert_session(db: &Db, token: &str, expires_at: &str) -> Result<(), DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    conn.execute(
        "INSERT INTO sessions (token, expires_at) VALUES (?1, ?2)",
        params![token, expires_at],
    )?;
    Ok(())
}

pub fn get_session_expiry(db: &Db, token: &str) -> Result<Option<String>, DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    match conn.query_row(
        "SELECT expires_at FROM sessions WHERE token = ?1",
        params![token],
        |row| row.get(0),
    ) {
        Ok(val) => Ok(Some(val)),
        Err(rusqlite::Error::QueryReturnedNoRows) => Ok(None),
        Err(e) => Err(DbError::Sqlite(e)),
    }
}

pub fn delete_session(db: &Db, token: &str) -> Result<(), DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    conn.execute("DELETE FROM sessions WHERE token = ?1", params![token])?;
    Ok(())
}

pub fn prune_expired_sessions(db: &Db) -> Result<(), DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    conn.execute(
        "DELETE FROM sessions WHERE expires_at < datetime('now')",
        [],
    )?;
    Ok(())
}
