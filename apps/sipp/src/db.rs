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

#[derive(Serialize, Deserialize)]
pub struct Snippet {
    pub id: i64,
    pub short_id: String,
    pub content: String,
    pub name: String,
}

fn generate_short_id() -> String {
    nanoid!(10)
}

pub fn db_path() -> String {
    std::env::var("SIPP_DB_PATH").unwrap_or_else(|_| "sipp.sqlite".to_string())
}

pub fn init_db() -> Result<Db, DbError> {
    let conn = Connection::open(db_path())?;
    conn.execute(
        "CREATE TABLE IF NOT EXISTS snippets (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            short_id TEXT NOT NULL UNIQUE,
            content TEXT NOT NULL,
            name TEXT NOT NULL
        )",
        [],
    )?;
    Ok(Arc::new(Mutex::new(conn)))
}

pub fn create_snippet(db: &Db, name: &str, content: &str) -> Result<Snippet, DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    let short_id = generate_short_id();
    conn.execute(
        "INSERT INTO snippets (short_id, content, name) VALUES (?1, ?2, ?3)",
        params![short_id, content, name],
    )?;
    let id = conn.last_insert_rowid();
    Ok(Snippet {
        id,
        short_id,
        content: content.to_string(),
        name: name.to_string(),
    })
}

pub fn get_snippet_by_short_id(db: &Db, short_id: &str) -> Result<Option<Snippet>, DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    match conn.query_row(
        "SELECT id, short_id, content, name FROM snippets WHERE short_id = ?1",
        params![short_id],
        |row| {
            Ok(Snippet {
                id: row.get(0)?,
                short_id: row.get(1)?,
                content: row.get(2)?,
                name: row.get(3)?,
            })
        },
    ) {
        Ok(snippet) => Ok(Some(snippet)),
        Err(rusqlite::Error::QueryReturnedNoRows) => Ok(None),
        Err(e) => Err(DbError::Sqlite(e)),
    }
}

pub fn get_all_snippets(db: &Db) -> Result<Vec<Snippet>, DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    let mut stmt = conn
        .prepare("SELECT id, short_id, content, name FROM snippets ORDER BY id DESC")?;
    let snippets = stmt.query_map([], |row| {
        Ok(Snippet {
            id: row.get(0)?,
            short_id: row.get(1)?,
            content: row.get(2)?,
            name: row.get(3)?,
        })
    })?
    .filter_map(|r| r.ok())
    .collect();
    Ok(snippets)
}

pub fn delete_snippet_by_short_id(db: &Db, short_id: &str) -> Result<bool, DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    let rows_affected = conn.execute(
        "DELETE FROM snippets WHERE short_id = ?1",
        params![short_id],
    )?;
    Ok(rows_affected > 0)
}

pub fn update_snippet_by_short_id(
    db: &Db,
    short_id: &str,
    name: &str,
    content: &str,
) -> Result<Option<Snippet>, DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    let rows_affected = conn.execute(
        "UPDATE snippets SET name = ?1, content = ?2 WHERE short_id = ?3",
        params![name, content, short_id],
    )?;
    if rows_affected == 0 {
        return Ok(None);
    }
    match conn.query_row(
        "SELECT id, short_id, content, name FROM snippets WHERE short_id = ?1",
        params![short_id],
        |row| {
            Ok(Snippet {
                id: row.get(0)?,
                short_id: row.get(1)?,
                content: row.get(2)?,
                name: row.get(3)?,
            })
        },
    ) {
        Ok(snippet) => Ok(Some(snippet)),
        Err(rusqlite::Error::QueryReturnedNoRows) => Ok(None),
        Err(e) => Err(DbError::Sqlite(e)),
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn test_db() -> Db {
        let conn = Connection::open_in_memory().unwrap();
        conn.execute(
            "CREATE TABLE IF NOT EXISTS snippets (
                id INTEGER PRIMARY KEY AUTOINCREMENT,
                short_id TEXT NOT NULL UNIQUE,
                content TEXT NOT NULL,
                name TEXT NOT NULL
            )",
            [],
        )
        .unwrap();
        Arc::new(Mutex::new(conn))
    }

    #[test]
    fn create_and_get_snippet() {
        let db = test_db();
        let snippet = create_snippet(&db, "hello.rs", "fn main() {}").unwrap();
        assert_eq!(snippet.name, "hello.rs");
        assert_eq!(snippet.content, "fn main() {}");
        assert!(!snippet.short_id.is_empty());

        let fetched = get_snippet_by_short_id(&db, &snippet.short_id)
            .unwrap()
            .unwrap();
        assert_eq!(fetched.name, "hello.rs");
        assert_eq!(fetched.content, "fn main() {}");
    }

    #[test]
    fn get_snippet_not_found() {
        let db = test_db();
        let result = get_snippet_by_short_id(&db, "nonexistent").unwrap();
        assert!(result.is_none());
    }

    #[test]
    fn get_all_snippets_ordered_desc() {
        let db = test_db();
        create_snippet(&db, "first", "aaa").unwrap();
        create_snippet(&db, "second", "bbb").unwrap();

        let all = get_all_snippets(&db).unwrap();
        assert_eq!(all.len(), 2);
        assert_eq!(all[0].name, "second"); // DESC order
        assert_eq!(all[1].name, "first");
    }

    #[test]
    fn update_snippet() {
        let db = test_db();
        let snippet = create_snippet(&db, "old.rs", "old content").unwrap();

        let updated = update_snippet_by_short_id(&db, &snippet.short_id, "new.rs", "new content")
            .unwrap()
            .unwrap();
        assert_eq!(updated.name, "new.rs");
        assert_eq!(updated.content, "new content");
    }

    #[test]
    fn update_nonexistent_snippet() {
        let db = test_db();
        let result = update_snippet_by_short_id(&db, "nope", "name", "content").unwrap();
        assert!(result.is_none());
    }

    #[test]
    fn delete_snippet() {
        let db = test_db();
        let snippet = create_snippet(&db, "test", "content").unwrap();

        let deleted = delete_snippet_by_short_id(&db, &snippet.short_id).unwrap();
        assert!(deleted);

        let fetched = get_snippet_by_short_id(&db, &snippet.short_id).unwrap();
        assert!(fetched.is_none());
    }

    #[test]
    fn delete_nonexistent_returns_false() {
        let db = test_db();
        let deleted = delete_snippet_by_short_id(&db, "nonexistent").unwrap();
        assert!(!deleted);
    }
}
