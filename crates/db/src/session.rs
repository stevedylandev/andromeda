use rusqlite::params;

use crate::{Db, DbError};

pub const SESSION_SCHEMA: &str = "
    CREATE TABLE IF NOT EXISTS sessions (
        id         INTEGER PRIMARY KEY AUTOINCREMENT,
        token      TEXT NOT NULL UNIQUE,
        expires_at TEXT NOT NULL
    );
";

pub fn insert_session(db: &Db, token: &str, expires_at: &str) -> Result<(), DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    conn.execute(
        "INSERT INTO sessions (token, expires_at) VALUES (?1, ?2)",
        params![token, expires_at],
    )?;
    Ok(())
}

pub fn get_session_expiry(db: &Db, token: &str) -> Result<Option<String>, DbError> {
    use rusqlite::OptionalExtension;
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    let val = conn
        .query_row(
            "SELECT expires_at FROM sessions WHERE token = ?1",
            params![token],
            |row| row.get(0),
        )
        .optional()?;
    Ok(val)
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

#[cfg(test)]
mod tests {
    use super::*;
    use rusqlite::Connection;
    use std::sync::{Arc, Mutex};

    fn test_db() -> Db {
        let conn = Connection::open_in_memory().unwrap();
        conn.execute_batch(SESSION_SCHEMA).unwrap();
        Arc::new(Mutex::new(conn))
    }

    #[test]
    fn session_lifecycle() {
        let db = test_db();
        insert_session(&db, "tok", "2099-12-31 23:59:59").unwrap();
        assert_eq!(
            get_session_expiry(&db, "tok").unwrap(),
            Some("2099-12-31 23:59:59".to_string())
        );
        delete_session(&db, "tok").unwrap();
        assert!(get_session_expiry(&db, "tok").unwrap().is_none());
    }

    #[test]
    fn prune_expired_sessions_works() {
        let db = test_db();
        insert_session(&db, "old", "2000-01-01 00:00:00").unwrap();
        insert_session(&db, "new", "2099-01-01 00:00:00").unwrap();
        prune_expired_sessions(&db).unwrap();
        assert!(get_session_expiry(&db, "old").unwrap().is_none());
        assert!(get_session_expiry(&db, "new").unwrap().is_some());
    }

    #[test]
    fn missing_token_returns_none() {
        let db = test_db();
        assert!(get_session_expiry(&db, "nonexistent").unwrap().is_none());
    }
}
