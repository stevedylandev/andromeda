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

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct Wine {
    pub id: i64,
    pub short_id: String,
    pub name: String,
    pub origin: String,
    pub grape: String,
    pub notes: String,
    pub has_image: bool,
    pub image_mime: Option<String>,
    pub sweetness: i32,
    pub acidity: i32,
    pub tannin: i32,
    pub alcohol: i32,
    pub body: i32,
    pub background: String,
    pub created_at: String,
}

pub fn init_db() -> Db {
    let path = std::env::var("CELLAR_DB_PATH").unwrap_or_else(|_| "cellar.sqlite".to_string());
    let conn = Connection::open(&path).expect("Failed to open database");

    conn.execute_batch(
        "CREATE TABLE IF NOT EXISTS wines (
            id         INTEGER PRIMARY KEY AUTOINCREMENT,
            short_id   TEXT NOT NULL UNIQUE,
            name       TEXT NOT NULL,
            origin     TEXT NOT NULL,
            grape      TEXT NOT NULL,
            notes      TEXT NOT NULL,
            image      BLOB,
            image_mime TEXT,
            sweetness  INTEGER NOT NULL CHECK(sweetness BETWEEN 1 AND 5),
            acidity    INTEGER NOT NULL CHECK(acidity BETWEEN 1 AND 5),
            tannin     INTEGER NOT NULL CHECK(tannin BETWEEN 1 AND 5),
            alcohol    INTEGER NOT NULL CHECK(alcohol BETWEEN 1 AND 5),
            body       INTEGER NOT NULL CHECK(body BETWEEN 1 AND 5),
            created_at TEXT NOT NULL DEFAULT (datetime('now'))
        );

        CREATE TABLE IF NOT EXISTS sessions (
            id         INTEGER PRIMARY KEY AUTOINCREMENT,
            token      TEXT NOT NULL UNIQUE,
            expires_at TEXT NOT NULL
        );"
    )
    .expect("Failed to create tables");

    // Migration: add background column if it doesn't exist
    let _ = conn.execute("ALTER TABLE wines ADD COLUMN background TEXT NOT NULL DEFAULT ''", []);

    Arc::new(Mutex::new(conn))
}

fn wine_from_row(row: &rusqlite::Row) -> rusqlite::Result<Wine> {
    Ok(Wine {
        id: row.get(0)?,
        short_id: row.get(1)?,
        name: row.get(2)?,
        origin: row.get(3)?,
        grape: row.get(4)?,
        notes: row.get(5)?,
        has_image: row.get(6)?,
        image_mime: row.get(7)?,
        sweetness: row.get(8)?,
        acidity: row.get(9)?,
        tannin: row.get(10)?,
        alcohol: row.get(11)?,
        body: row.get(12)?,
        background: row.get(13)?,
        created_at: row.get(14)?,
    })
}

const WINE_COLUMNS: &str =
    "id, short_id, name, origin, grape, notes, (image IS NOT NULL) AS has_image, image_mime, sweetness, acidity, tannin, alcohol, body, background, created_at";

pub fn create_wine(
    db: &Db,
    name: &str,
    origin: &str,
    grape: &str,
    notes: &str,
    image: Option<&[u8]>,
    image_mime: Option<&str>,
    sweetness: i32,
    acidity: i32,
    tannin: i32,
    alcohol: i32,
    body: i32,
    background: &str,
) -> Result<Wine, DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    let short_id = nanoid!(10);
    conn.execute(
        "INSERT INTO wines (short_id, name, origin, grape, notes, image, image_mime, sweetness, acidity, tannin, alcohol, body, background)
         VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7, ?8, ?9, ?10, ?11, ?12, ?13)",
        params![short_id, name, origin, grape, notes, image, image_mime, sweetness, acidity, tannin, alcohol, body, background],
    )?;
    let id = conn.last_insert_rowid();
    let wine = conn.query_row(
        &format!("SELECT {} FROM wines WHERE id = ?1", WINE_COLUMNS),
        params![id],
        wine_from_row,
    )?;
    Ok(wine)
}

pub fn get_all_wines(db: &Db) -> Result<Vec<Wine>, DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    let mut stmt = conn.prepare(&format!(
        "SELECT {} FROM wines ORDER BY id DESC",
        WINE_COLUMNS
    ))?;
    let wines = stmt
        .query_map([], wine_from_row)?
        .collect::<Result<Vec<_>, _>>()?;
    Ok(wines)
}

pub fn get_wine_by_short_id(db: &Db, short_id: &str) -> Result<Option<Wine>, DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    match conn.query_row(
        &format!(
            "SELECT {} FROM wines WHERE short_id = ?1",
            WINE_COLUMNS
        ),
        params![short_id],
        wine_from_row,
    ) {
        Ok(wine) => Ok(Some(wine)),
        Err(rusqlite::Error::QueryReturnedNoRows) => Ok(None),
        Err(e) => Err(DbError::Sqlite(e)),
    }
}

pub fn get_wine_image(db: &Db, short_id: &str) -> Result<Option<(Vec<u8>, String)>, DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    match conn.query_row(
        "SELECT image, image_mime FROM wines WHERE short_id = ?1 AND image IS NOT NULL",
        params![short_id],
        |row| {
            let image: Vec<u8> = row.get(0)?;
            let mime: String = row.get(1)?;
            Ok((image, mime))
        },
    ) {
        Ok(result) => Ok(Some(result)),
        Err(rusqlite::Error::QueryReturnedNoRows) => Ok(None),
        Err(e) => Err(DbError::Sqlite(e)),
    }
}

pub fn update_wine(
    db: &Db,
    short_id: &str,
    name: &str,
    origin: &str,
    grape: &str,
    notes: &str,
    sweetness: i32,
    acidity: i32,
    tannin: i32,
    alcohol: i32,
    body: i32,
    background: &str,
) -> Result<Option<Wine>, DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    let rows = conn.execute(
        "UPDATE wines SET name = ?1, origin = ?2, grape = ?3, notes = ?4, sweetness = ?5, acidity = ?6, tannin = ?7, alcohol = ?8, body = ?9, background = ?10 WHERE short_id = ?11",
        params![name, origin, grape, notes, sweetness, acidity, tannin, alcohol, body, background, short_id],
    )?;
    if rows == 0 {
        return Ok(None);
    }
    match conn.query_row(
        &format!(
            "SELECT {} FROM wines WHERE short_id = ?1",
            WINE_COLUMNS
        ),
        params![short_id],
        wine_from_row,
    ) {
        Ok(wine) => Ok(Some(wine)),
        Err(rusqlite::Error::QueryReturnedNoRows) => Ok(None),
        Err(e) => Err(DbError::Sqlite(e)),
    }
}

pub fn update_wine_image(
    db: &Db,
    short_id: &str,
    image: &[u8],
    mime: &str,
) -> Result<bool, DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    let rows = conn.execute(
        "UPDATE wines SET image = ?1, image_mime = ?2 WHERE short_id = ?3",
        params![image, mime, short_id],
    )?;
    Ok(rows > 0)
}

pub fn delete_wine(db: &Db, short_id: &str) -> Result<bool, DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    let rows = conn.execute(
        "DELETE FROM wines WHERE short_id = ?1",
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
