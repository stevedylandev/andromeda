use rusqlite::{Connection, OptionalExtension, params};
use std::sync::{Arc, Mutex};

pub use andromeda_db::{Db, DbError};
pub use andromeda_db::session::{insert_session, get_session_expiry, delete_session, prune_expired_sessions};

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

const SCHEMA: &str = "
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
";

pub fn init_db() -> Db {
    let path = "parcels.db";
    let conn = Connection::open(path).expect("Failed to open database");
    conn.execute_batch("PRAGMA journal_mode=WAL; PRAGMA foreign_keys=ON;")
        .expect("Failed to set PRAGMAs");
    conn.execute_batch(SCHEMA).expect("Failed to create tables");
    Arc::new(Mutex::new(conn))
}

// ── Row Helpers ────────────────────────────────────────────────────────────

const PACKAGE_COLS: &str = "id, tracking_number, label, status, status_category, status_summary, mail_class, expected_delivery, last_refreshed_at, created_at";

fn package_from_row(row: &rusqlite::Row) -> rusqlite::Result<Package> {
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
}

const EVENT_COLS: &str = "id, package_id, event_timestamp, event_type, event_city, event_state, event_zip, event_code";

fn event_from_row(row: &rusqlite::Row) -> rusqlite::Result<TrackingEvent> {
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
}

// ── Package Queries ─────────────────────────────────────────────────────────

pub fn list_packages(db: &Db) -> Result<Vec<Package>, DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    let mut stmt = conn.prepare(
        &format!("SELECT {} FROM packages ORDER BY created_at DESC", PACKAGE_COLS),
    )?;
    let packages = stmt
        .query_map([], package_from_row)?
        .collect::<Result<Vec<_>, _>>()?;
    Ok(packages)
}

pub fn get_package(db: &Db, id: i64) -> Result<Option<Package>, DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    let pkg = conn
        .query_row(
            &format!("SELECT {} FROM packages WHERE id = ?1", PACKAGE_COLS),
            params![id],
            package_from_row,
        )
        .optional()?;
    Ok(pkg)
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
        &format!("SELECT {} FROM tracking_events WHERE package_id = ?1 ORDER BY event_timestamp DESC", EVENT_COLS),
    )?;
    let events = stmt
        .query_map(params![package_id], event_from_row)?
        .collect::<Result<Vec<_>, _>>()?;
    Ok(events)
}

#[cfg(test)]
mod tests {
    use super::*;

    fn test_db() -> Db {
        let conn = Connection::open_in_memory().unwrap();
        conn.execute_batch("PRAGMA foreign_keys=ON;").unwrap();
        conn.execute_batch(SCHEMA).unwrap();
        Arc::new(Mutex::new(conn))
    }

    // ── Package CRUD ───────────────────────────────────────────────────

    #[test]
    fn insert_and_list_packages() {
        let db = test_db();
        let id = insert_package(&db, "TRACK001", Some("My Package")).unwrap();
        assert!(id > 0);

        let packages = list_packages(&db).unwrap();
        assert_eq!(packages.len(), 1);
        assert_eq!(packages[0].tracking_number, "TRACK001");
        assert_eq!(packages[0].label.as_deref(), Some("My Package"));
    }

    #[test]
    fn insert_duplicate_tracking_number_fails() {
        let db = test_db();
        insert_package(&db, "TRACK001", None).unwrap();
        let result = insert_package(&db, "TRACK001", None);
        assert!(result.is_err());
    }

    #[test]
    fn get_package_found() {
        let db = test_db();
        let id = insert_package(&db, "TRACK001", None).unwrap();
        let pkg = get_package(&db, id).unwrap();
        assert!(pkg.is_some());
        assert_eq!(pkg.unwrap().tracking_number, "TRACK001");
    }

    #[test]
    fn get_package_not_found() {
        let db = test_db();
        let pkg = get_package(&db, 999).unwrap();
        assert!(pkg.is_none());
    }

    #[test]
    fn update_package_status_works() {
        let db = test_db();
        let id = insert_package(&db, "TRACK001", None).unwrap();
        update_package_status(
            &db,
            id,
            "Delivered",
            Some("Delivered"),
            Some("Package delivered"),
            Some("Priority"),
            Some("2024-01-20"),
            "2024-01-18 12:00:00",
        )
        .unwrap();

        let pkg = get_package(&db, id).unwrap().unwrap();
        assert_eq!(pkg.status.as_deref(), Some("Delivered"));
        assert_eq!(pkg.mail_class.as_deref(), Some("Priority"));
    }

    #[test]
    fn delete_package_removes_it() {
        let db = test_db();
        let id = insert_package(&db, "TRACK001", None).unwrap();
        delete_package(&db, id).unwrap();
        assert!(get_package(&db, id).unwrap().is_none());
    }

    // ── Tracking Events ────────────────────────────────────────────────

    #[test]
    fn insert_and_get_events() {
        let db = test_db();
        let pkg_id = insert_package(&db, "TRACK001", None).unwrap();

        insert_event(
            &db,
            pkg_id,
            Some("2024-01-15 10:00:00"),
            Some("Delivered"),
            Some("New York"),
            Some("NY"),
            Some("10001"),
            Some("01"),
        )
        .unwrap();

        insert_event(
            &db,
            pkg_id,
            Some("2024-01-14 08:00:00"),
            Some("In Transit"),
            Some("Chicago"),
            Some("IL"),
            None,
            None,
        )
        .unwrap();

        let events = get_events_for_package(&db, pkg_id).unwrap();
        assert_eq!(events.len(), 2);
        // Ordered by timestamp DESC
        assert_eq!(events[0].event_city.as_deref(), Some("New York"));
        assert_eq!(events[1].event_city.as_deref(), Some("Chicago"));
    }

    #[test]
    fn delete_events_for_package_clears_them() {
        let db = test_db();
        let pkg_id = insert_package(&db, "TRACK001", None).unwrap();
        insert_event(&db, pkg_id, None, Some("Shipped"), None, None, None, None).unwrap();

        delete_events_for_package(&db, pkg_id).unwrap();
        let events = get_events_for_package(&db, pkg_id).unwrap();
        assert!(events.is_empty());
    }

    #[test]
    fn delete_package_cascades_to_events() {
        let db = test_db();
        let pkg_id = insert_package(&db, "TRACK001", None).unwrap();
        insert_event(&db, pkg_id, None, Some("Shipped"), None, None, None, None).unwrap();

        delete_package(&db, pkg_id).unwrap();
        let events = get_events_for_package(&db, pkg_id).unwrap();
        assert!(events.is_empty());
    }

    // ── Sessions ───────────────────────────────────────────────────────

    #[test]
    fn session_crud_lifecycle() {
        let db = test_db();
        let token = "abc123";

        insert_session(&db, token, "2099-01-01 00:00:00").unwrap();

        let expiry = get_session_expiry(&db, token).unwrap();
        assert_eq!(expiry, Some("2099-01-01 00:00:00".to_string()));

        delete_session(&db, token).unwrap();
        let expiry = get_session_expiry(&db, token).unwrap();
        assert!(expiry.is_none());
    }

    #[test]
    fn get_session_expiry_missing_token() {
        let db = test_db();
        let expiry = get_session_expiry(&db, "nonexistent").unwrap();
        assert!(expiry.is_none());
    }

    #[test]
    fn prune_expired_sessions_removes_old() {
        let db = test_db();
        insert_session(&db, "expired", "2000-01-01 00:00:00").unwrap();
        insert_session(&db, "valid", "2099-01-01 00:00:00").unwrap();

        prune_expired_sessions(&db).unwrap();

        assert!(get_session_expiry(&db, "expired").unwrap().is_none());
        assert!(get_session_expiry(&db, "valid").unwrap().is_some());
    }
}
