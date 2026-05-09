use rusqlite::{params, Connection, OptionalExtension};
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

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DailyArtwork {
    pub date: String,
    pub artwork_id: i64,
    pub title: String,
    pub artist_display: Option<String>,
    pub artist_title: Option<String>,
    pub date_display: Option<String>,
    pub medium_display: Option<String>,
    pub dimensions: Option<String>,
    pub place_of_origin: Option<String>,
    pub credit_line: Option<String>,
    pub description: Option<String>,
    pub short_description: Option<String>,
    pub image_id: String,
    pub fetched_at: String,
}

const SCHEMA: &str = "
    CREATE TABLE IF NOT EXISTS daily_artworks (
        date              TEXT PRIMARY KEY,
        artwork_id        INTEGER NOT NULL,
        title             TEXT NOT NULL,
        artist_display    TEXT,
        artist_title      TEXT,
        date_display      TEXT,
        medium_display    TEXT,
        dimensions        TEXT,
        place_of_origin   TEXT,
        credit_line       TEXT,
        description       TEXT,
        short_description TEXT,
        image_id          TEXT NOT NULL,
        fetched_at        TEXT NOT NULL DEFAULT (datetime('now'))
    );
    CREATE INDEX IF NOT EXISTS idx_daily_artworks_artwork_id ON daily_artworks(artwork_id);
";

pub fn init_db(path: &str) -> Db {
    let conn = Connection::open(path).expect("Failed to open easel database");
    conn.execute_batch(SCHEMA).expect("Failed to apply schema");
    Arc::new(Mutex::new(conn))
}

const COLS: &str = "date, artwork_id, title, artist_display, artist_title, date_display, medium_display, dimensions, place_of_origin, credit_line, description, short_description, image_id, fetched_at";

fn from_row(row: &rusqlite::Row) -> rusqlite::Result<DailyArtwork> {
    Ok(DailyArtwork {
        date: row.get(0)?,
        artwork_id: row.get(1)?,
        title: row.get(2)?,
        artist_display: row.get(3)?,
        artist_title: row.get(4)?,
        date_display: row.get(5)?,
        medium_display: row.get(6)?,
        dimensions: row.get(7)?,
        place_of_origin: row.get(8)?,
        credit_line: row.get(9)?,
        description: row.get(10)?,
        short_description: row.get(11)?,
        image_id: row.get(12)?,
        fetched_at: row.get(13)?,
    })
}

pub fn insert_daily(db: &Db, art: &DailyArtwork) -> Result<bool, DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    let rows = conn.execute(
        "INSERT OR IGNORE INTO daily_artworks
         (date, artwork_id, title, artist_display, artist_title, date_display, medium_display,
          dimensions, place_of_origin, credit_line, description, short_description, image_id, fetched_at)
         VALUES (?1,?2,?3,?4,?5,?6,?7,?8,?9,?10,?11,?12,?13,?14)",
        params![
            art.date,
            art.artwork_id,
            art.title,
            art.artist_display,
            art.artist_title,
            art.date_display,
            art.medium_display,
            art.dimensions,
            art.place_of_origin,
            art.credit_line,
            art.description,
            art.short_description,
            art.image_id,
            art.fetched_at,
        ],
    )?;
    Ok(rows > 0)
}

pub fn get_daily(db: &Db, date: &str) -> Result<Option<DailyArtwork>, DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    let row = conn
        .query_row(
            &format!("SELECT {COLS} FROM daily_artworks WHERE date = ?1"),
            params![date],
            from_row,
        )
        .optional()?;
    Ok(row)
}

pub fn list_daily(db: &Db, limit: i64) -> Result<Vec<DailyArtwork>, DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    let mut stmt = conn.prepare(&format!(
        "SELECT {COLS} FROM daily_artworks ORDER BY date DESC LIMIT ?1"
    ))?;
    let rows = stmt
        .query_map(params![limit], from_row)?
        .collect::<Result<Vec<_>, _>>()?;
    Ok(rows)
}

pub fn artwork_id_exists(db: &Db, artwork_id: i64) -> Result<bool, DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    let count: i64 = conn.query_row(
        "SELECT COUNT(*) FROM daily_artworks WHERE artwork_id = ?1",
        params![artwork_id],
        |row| row.get(0),
    )?;
    Ok(count > 0)
}

pub fn missing_dates(db: &Db, dates: &[String]) -> Result<Vec<String>, DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    let mut missing = Vec::new();
    for d in dates {
        let exists: i64 = conn.query_row(
            "SELECT COUNT(*) FROM daily_artworks WHERE date = ?1",
            params![d],
            |row| row.get(0),
        )?;
        if exists == 0 {
            missing.push(d.clone());
        }
    }
    Ok(missing)
}

#[cfg(test)]
mod tests {
    use super::*;

    fn test_db() -> Db {
        let conn = Connection::open_in_memory().unwrap();
        conn.execute_batch(SCHEMA).unwrap();
        Arc::new(Mutex::new(conn))
    }

    fn sample(date: &str, artwork_id: i64) -> DailyArtwork {
        DailyArtwork {
            date: date.to_string(),
            artwork_id,
            title: "Test".to_string(),
            artist_display: Some("An Artist".to_string()),
            artist_title: None,
            date_display: None,
            medium_display: None,
            dimensions: None,
            place_of_origin: None,
            credit_line: None,
            description: None,
            short_description: None,
            image_id: "abc-123".to_string(),
            fetched_at: "2024-01-01T00:00:00Z".to_string(),
        }
    }

    #[test]
    fn insert_and_get() {
        let db = test_db();
        assert!(insert_daily(&db, &sample("2024-01-01", 1)).unwrap());
        let got = get_daily(&db, "2024-01-01").unwrap().unwrap();
        assert_eq!(got.artwork_id, 1);
    }

    #[test]
    fn duplicate_date_ignored() {
        let db = test_db();
        assert!(insert_daily(&db, &sample("2024-01-01", 1)).unwrap());
        assert!(!insert_daily(&db, &sample("2024-01-01", 2)).unwrap());
        assert_eq!(get_daily(&db, "2024-01-01").unwrap().unwrap().artwork_id, 1);
    }

    #[test]
    fn artwork_id_exists_works() {
        let db = test_db();
        insert_daily(&db, &sample("2024-01-01", 42)).unwrap();
        assert!(artwork_id_exists(&db, 42).unwrap());
        assert!(!artwork_id_exists(&db, 99).unwrap());
    }

    #[test]
    fn missing_dates_filter() {
        let db = test_db();
        insert_daily(&db, &sample("2024-01-01", 1)).unwrap();
        let dates = vec![
            "2024-01-01".to_string(),
            "2024-01-02".to_string(),
            "2024-01-03".to_string(),
        ];
        let missing = missing_dates(&db, &dates).unwrap();
        assert_eq!(missing, vec!["2024-01-02", "2024-01-03"]);
    }

    #[test]
    fn list_daily_desc() {
        let db = test_db();
        insert_daily(&db, &sample("2024-01-01", 1)).unwrap();
        insert_daily(&db, &sample("2024-01-03", 3)).unwrap();
        insert_daily(&db, &sample("2024-01-02", 2)).unwrap();
        let list = list_daily(&db, 10).unwrap();
        assert_eq!(list.len(), 3);
        assert_eq!(list[0].date, "2024-01-03");
        assert_eq!(list[2].date, "2024-01-01");
    }
}
