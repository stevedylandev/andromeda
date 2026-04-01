use rusqlite::{Connection, params};
use std::fmt;
use std::sync::{Arc, Mutex};

pub type Db = Arc<Mutex<Connection>>;

// ── Error ───────────────────────────────────────────────────────────────────

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

// ── Types ───────────────────────────────────────────────────────────────────

#[derive(Debug, Clone)]
pub struct Package {
    pub id: i64,
    pub tracking_number: String,
    pub label: Option<String>,
    pub status: Option<String>,
    pub status_category: Option<String>,
    pub status_summary: Option<String>,
    pub mail_class: Option<String>,
    pub expected_delivery: Option<String>,
    pub last_refreshed_at: Option<String>,
    pub created_at: String,
}

#[derive(Debug, Clone)]
pub struct TrackingEvent {
    pub id: i64,
    pub package_id: i64,
    pub event_timestamp: Option<String>,
    pub event_type: Option<String>,
    pub event_city: Option<String>,
    pub event_state: Option<String>,
    pub event_zip: Option<String>,
    pub event_code: Option<String>,
}

// ── Pool Setup ──────────────────────────────────────────────────────────────

pub fn database_path() -> String {
    let raw = std::env::var("DATABASE_URL").unwrap_or_else(|_| "parcels.db".to_string());
    // Strip sqlite:// or sqlite:/// prefix if present
    raw.strip_prefix("sqlite:///")
        .or_else(|| raw.strip_prefix("sqlite://"))
        .unwrap_or(&raw)
        .to_string()
}

pub fn init_db(path: &str) -> Result<Db, DbError> {
    if let Some(parent) = std::path::Path::new(path).parent() {
        if parent != std::path::Path::new("") {
            std::fs::create_dir_all(parent).ok();
        }
    }
    let conn = Connection::open(path)?;
    conn.execute_batch("PRAGMA journal_mode=WAL; PRAGMA foreign_keys=ON;")?;
    conn.execute_batch("
        CREATE TABLE IF NOT EXISTS packages (
            id                INTEGER PRIMARY KEY AUTOINCREMENT,
            tracking_number   TEXT NOT NULL UNIQUE,
            label             TEXT,
            status            TEXT,
            status_category   TEXT,
            status_summary    TEXT,
            mail_class        TEXT,
            expected_delivery TEXT,
            last_refreshed_at TEXT,
            created_at        TEXT NOT NULL DEFAULT (datetime('now'))
        );
        CREATE TABLE IF NOT EXISTS tracking_events (
            id              INTEGER PRIMARY KEY AUTOINCREMENT,
            package_id      INTEGER NOT NULL REFERENCES packages(id) ON DELETE CASCADE,
            event_timestamp TEXT,
            event_type      TEXT,
            event_city      TEXT,
            event_state     TEXT,
            event_zip       TEXT,
            event_code      TEXT
        );
        CREATE TABLE IF NOT EXISTS sessions (
            id         INTEGER PRIMARY KEY AUTOINCREMENT,
            token      TEXT NOT NULL UNIQUE,
            expires_at TEXT NOT NULL
        );
    ")?;
    Ok(Arc::new(Mutex::new(conn)))
}

// ── Package Queries ─────────────────────────────────────────────────────────

pub fn list_packages(db: &Db) -> Result<Vec<Package>, DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    let mut stmt = conn.prepare(
        "SELECT id, tracking_number, label, status, status_category, status_summary,
                mail_class, expected_delivery, last_refreshed_at, created_at
         FROM packages ORDER BY created_at DESC",
    )?;
    let packages = stmt
        .query_map([], |row| {
            Ok(Package {
                id: row.get(0)?,
                tracking_number: row.get(1)?,
                label: row.get(2)?,
                status: row.get(3)?,
                status_category: row.get(4)?,
                status_summary: row.get(5)?,
                mail_class: row.get(6)?,
                expected_delivery: row.get(7)?,
                last_refreshed_at: row.get(8)?,
                created_at: row.get(9)?,
            })
        })?
        .filter_map(|r| r.ok())
        .collect();
    Ok(packages)
}

pub fn get_package(db: &Db, id: i64) -> Result<Option<Package>, DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    match conn.query_row(
        "SELECT id, tracking_number, label, status, status_category, status_summary,
                mail_class, expected_delivery, last_refreshed_at, created_at
         FROM packages WHERE id = ?1",
        params![id],
        |row| {
            Ok(Package {
                id: row.get(0)?,
                tracking_number: row.get(1)?,
                label: row.get(2)?,
                status: row.get(3)?,
                status_category: row.get(4)?,
                status_summary: row.get(5)?,
                mail_class: row.get(6)?,
                expected_delivery: row.get(7)?,
                last_refreshed_at: row.get(8)?,
                created_at: row.get(9)?,
            })
        },
    ) {
        Ok(pkg) => Ok(Some(pkg)),
        Err(rusqlite::Error::QueryReturnedNoRows) => Ok(None),
        Err(e) => Err(DbError::Sqlite(e)),
    }
}

pub fn insert_package(db: &Db, tracking_number: &str, label: Option<&str>) -> Result<i64, DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    conn.execute(
        "INSERT INTO packages (tracking_number, label) VALUES (?1, ?2)",
        params![tracking_number, label],
    )?;
    Ok(conn.last_insert_rowid())
}

pub fn update_package_status(
    db: &Db,
    id: i64,
    status: &str,
    status_category: Option<&str>,
    status_summary: Option<&str>,
    mail_class: Option<&str>,
    expected_delivery: Option<&str>,
    last_refreshed_at: &str,
) -> Result<(), DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    conn.execute(
        "UPDATE packages SET status = ?1, status_category = ?2, status_summary = ?3,
         mail_class = ?4, expected_delivery = ?5, last_refreshed_at = ?6
         WHERE id = ?7",
        params![status, status_category, status_summary, mail_class, expected_delivery, last_refreshed_at, id],
    )?;
    Ok(())
}

pub fn delete_package(db: &Db, id: i64) -> Result<(), DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    conn.execute("DELETE FROM packages WHERE id = ?1", params![id])?;
    Ok(())
}

// ── Tracking Event Queries ───────────────────────────────────────────────────

pub fn delete_events_for_package(db: &Db, package_id: i64) -> Result<(), DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    conn.execute("DELETE FROM tracking_events WHERE package_id = ?1", params![package_id])?;
    Ok(())
}

pub fn insert_event(
    db: &Db,
    package_id: i64,
    event_timestamp: Option<&str>,
    event_type: Option<&str>,
    event_city: Option<&str>,
    event_state: Option<&str>,
    event_zip: Option<&str>,
    event_code: Option<&str>,
) -> Result<(), DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    conn.execute(
        "INSERT INTO tracking_events
         (package_id, event_timestamp, event_type, event_city, event_state, event_zip, event_code)
         VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7)",
        params![package_id, event_timestamp, event_type, event_city, event_state, event_zip, event_code],
    )?;
    Ok(())
}

pub fn get_events_for_package(db: &Db, package_id: i64) -> Result<Vec<TrackingEvent>, DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    let mut stmt = conn.prepare(
        "SELECT id, package_id, event_timestamp, event_type, event_city, event_state,
                event_zip, event_code
         FROM tracking_events WHERE package_id = ?1
         ORDER BY event_timestamp DESC",
    )?;
    let events = stmt
        .query_map(params![package_id], |row| {
            Ok(TrackingEvent {
                id: row.get(0)?,
                package_id: row.get(1)?,
                event_timestamp: row.get(2)?,
                event_type: row.get(3)?,
                event_city: row.get(4)?,
                event_state: row.get(5)?,
                event_zip: row.get(6)?,
                event_code: row.get(7)?,
            })
        })?
        .filter_map(|r| r.ok())
        .collect();
    Ok(events)
}

// ── Session Queries ──────────────────────────────────────────────────────────

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
        Ok(expires_at) => Ok(Some(expires_at)),
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
    conn.execute("DELETE FROM sessions WHERE expires_at < datetime('now')", [])?;
    Ok(())
}
