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
    pub clarity: i32,
    pub color_intensity: i32,
    pub aroma_intensity: i32,
    pub nose_complexity: i32,
    pub background: String,
    pub created_at: String,
    pub wishlist: bool,
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

    // Migration: add appearance and nose tasting attributes
    let _ = conn.execute("ALTER TABLE wines ADD COLUMN clarity INTEGER NOT NULL DEFAULT 3", []);
    let _ = conn.execute("ALTER TABLE wines ADD COLUMN color_intensity INTEGER NOT NULL DEFAULT 3", []);
    let _ = conn.execute("ALTER TABLE wines ADD COLUMN aroma_intensity INTEGER NOT NULL DEFAULT 3", []);
    let _ = conn.execute("ALTER TABLE wines ADD COLUMN nose_complexity INTEGER NOT NULL DEFAULT 3", []);

    // Migration: add wishlist flag
    let _ = conn.execute("ALTER TABLE wines ADD COLUMN wishlist INTEGER NOT NULL DEFAULT 0", []);

    Arc::new(Mutex::new(conn))
}

fn wine_from_row(row: &rusqlite::Row) -> rusqlite::Result<Wine> {
    serde_rusqlite::from_row::<Wine>(row).map_err(|e| {
        rusqlite::Error::FromSqlConversionFailure(0, rusqlite::types::Type::Null, Box::new(e))
    })
}

const WINE_COLUMNS: &str =
    "id, short_id, name, origin, grape, notes, (image IS NOT NULL) AS has_image, image_mime, sweetness, acidity, tannin, alcohol, body, clarity, color_intensity, aroma_intensity, nose_complexity, background, created_at, wishlist";

#[derive(Serialize)]
pub struct WineInput<'a> {
    pub name: &'a str,
    pub origin: &'a str,
    pub grape: &'a str,
    pub notes: &'a str,
    pub sweetness: i32,
    pub acidity: i32,
    pub tannin: i32,
    pub alcohol: i32,
    pub body: i32,
    pub clarity: i32,
    pub color_intensity: i32,
    pub aroma_intensity: i32,
    pub nose_complexity: i32,
    pub background: &'a str,
}

pub fn create_wine(db: &Db, input: &WineInput, wishlist: bool) -> Result<Wine, DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    let short_id = nanoid!(10);
    let named = serde_rusqlite::to_params_named(input)
        .map_err(|e| rusqlite::Error::ToSqlConversionFailure(Box::new(e)))?;
    let mut bindings = named.to_slice();
    bindings.push((":short_id", &short_id));
    bindings.push((":wishlist", &wishlist));
    conn.execute(
        "INSERT INTO wines (short_id, name, origin, grape, notes, sweetness, acidity, tannin, alcohol, body, clarity, color_intensity, aroma_intensity, nose_complexity, background, wishlist)
         VALUES (:short_id, :name, :origin, :grape, :notes, :sweetness, :acidity, :tannin, :alcohol, :body, :clarity, :color_intensity, :aroma_intensity, :nose_complexity, :background, :wishlist)",
        bindings.as_slice(),
    )?;
    let id = conn.last_insert_rowid();
    let wine = conn.query_row(
        &format!("SELECT {} FROM wines WHERE id = ?1", WINE_COLUMNS),
        params![id],
        wine_from_row,
    )?;
    Ok(wine)
}

pub fn get_cellar_wines(db: &Db) -> Result<Vec<Wine>, DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    let mut stmt = conn.prepare(&format!(
        "SELECT {} FROM wines WHERE wishlist = 0 ORDER BY id DESC",
        WINE_COLUMNS
    ))?;
    let wines = stmt
        .query_map([], wine_from_row)?
        .collect::<Result<Vec<_>, _>>()?;
    Ok(wines)
}

pub fn get_wishlist_wines(db: &Db) -> Result<Vec<Wine>, DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    let mut stmt = conn.prepare(&format!(
        "SELECT {} FROM wines WHERE wishlist = 1 ORDER BY id DESC",
        WINE_COLUMNS
    ))?;
    let wines = stmt
        .query_map([], wine_from_row)?
        .collect::<Result<Vec<_>, _>>()?;
    Ok(wines)
}

pub fn promote_wine(db: &Db, short_id: &str) -> Result<bool, DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    let rows = conn.execute(
        "UPDATE wines SET wishlist = 0 WHERE short_id = ?1 AND wishlist = 1",
        params![short_id],
    )?;
    Ok(rows > 0)
}

pub fn update_wishlist_wine(
    db: &Db,
    short_id: &str,
    name: &str,
    origin: &str,
    grape: &str,
    notes: &str,
    background: &str,
) -> Result<Option<Wine>, DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    let rows = conn.execute(
        "UPDATE wines SET name = ?1, origin = ?2, grape = ?3, notes = ?4, background = ?5 WHERE short_id = ?6 AND wishlist = 1",
        params![name, origin, grape, notes, background, short_id],
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
    input: &WineInput,
) -> Result<Option<Wine>, DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    let named = serde_rusqlite::to_params_named(input)
        .map_err(|e| rusqlite::Error::ToSqlConversionFailure(Box::new(e)))?;
    let mut bindings = named.to_slice();
    bindings.push((":short_id", &short_id));
    let rows = conn.execute(
        "UPDATE wines SET name = :name, origin = :origin, grape = :grape, notes = :notes, sweetness = :sweetness, acidity = :acidity, tannin = :tannin, alcohol = :alcohol, body = :body, clarity = :clarity, color_intensity = :color_intensity, aroma_intensity = :aroma_intensity, nose_complexity = :nose_complexity, background = :background WHERE short_id = :short_id",
        bindings.as_slice(),
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

#[cfg(test)]
mod tests {
    use super::*;

    fn test_db() -> Db {
        let conn = Connection::open_in_memory().unwrap();
        conn.execute_batch(
            "CREATE TABLE IF NOT EXISTS wines (
                id              INTEGER PRIMARY KEY AUTOINCREMENT,
                short_id        TEXT NOT NULL UNIQUE,
                name            TEXT NOT NULL,
                origin          TEXT NOT NULL,
                grape           TEXT NOT NULL,
                notes           TEXT NOT NULL,
                image           BLOB,
                image_mime      TEXT,
                sweetness       INTEGER NOT NULL CHECK(sweetness BETWEEN 1 AND 5),
                acidity         INTEGER NOT NULL CHECK(acidity BETWEEN 1 AND 5),
                tannin          INTEGER NOT NULL CHECK(tannin BETWEEN 1 AND 5),
                alcohol         INTEGER NOT NULL CHECK(alcohol BETWEEN 1 AND 5),
                body            INTEGER NOT NULL CHECK(body BETWEEN 1 AND 5),
                clarity         INTEGER NOT NULL DEFAULT 3,
                color_intensity INTEGER NOT NULL DEFAULT 3,
                aroma_intensity INTEGER NOT NULL DEFAULT 3,
                nose_complexity INTEGER NOT NULL DEFAULT 3,
                background      TEXT NOT NULL DEFAULT '',
                created_at      TEXT NOT NULL DEFAULT (datetime('now')),
                wishlist        INTEGER NOT NULL DEFAULT 0
            );
            CREATE TABLE IF NOT EXISTS sessions (
                id         INTEGER PRIMARY KEY AUTOINCREMENT,
                token      TEXT NOT NULL UNIQUE,
                expires_at TEXT NOT NULL
            );",
        )
        .unwrap();
        Arc::new(Mutex::new(conn))
    }

    fn sample_input<'a>(name: &'a str, sweetness: i32) -> WineInput<'a> {
        WineInput {
            name,
            origin: "France",
            grape: "Merlot",
            notes: "Smooth",
            sweetness,
            acidity: 3,
            tannin: 3,
            alcohol: 3,
            body: 3,
            clarity: 3,
            color_intensity: 3,
            aroma_intensity: 3,
            nose_complexity: 3,
            background: "",
        }
    }

    fn create_test_wine(db: &Db, name: &str, wishlist: bool) -> Wine {
        create_wine(db, &sample_input(name, 3), wishlist).unwrap()
    }

    // ── Wine CRUD ──────────────────────────────────────────────────────

    #[test]
    fn create_and_get_wine() {
        let db = test_db();
        let wine = create_test_wine(&db, "Chateau Test", false);
        assert_eq!(wine.name, "Chateau Test");
        assert_eq!(wine.origin, "France");
        assert!(!wine.wishlist);

        let fetched = get_wine_by_short_id(&db, &wine.short_id).unwrap().unwrap();
        assert_eq!(fetched.name, "Chateau Test");
    }

    #[test]
    fn create_wine_invalid_sweetness_fails() {
        let db = test_db();
        let result = create_wine(&db, &sample_input("Bad", 6), false);
        assert!(result.is_err());
    }

    #[test]
    fn create_wine_zero_rating_fails() {
        let db = test_db();
        let result = create_wine(&db, &sample_input("Bad", 0), false);
        assert!(result.is_err());
    }

    #[test]
    fn get_cellar_wines_excludes_wishlist() {
        let db = test_db();
        create_test_wine(&db, "Cellar Wine", false);
        create_test_wine(&db, "Wishlist Wine", true);

        let cellar = get_cellar_wines(&db).unwrap();
        assert_eq!(cellar.len(), 1);
        assert_eq!(cellar[0].name, "Cellar Wine");
    }

    #[test]
    fn get_wishlist_wines_only_wishlist() {
        let db = test_db();
        create_test_wine(&db, "Cellar Wine", false);
        create_test_wine(&db, "Wishlist Wine", true);

        let wishlist = get_wishlist_wines(&db).unwrap();
        assert_eq!(wishlist.len(), 1);
        assert_eq!(wishlist[0].name, "Wishlist Wine");
    }

    #[test]
    fn promote_wine_moves_to_cellar() {
        let db = test_db();
        let wine = create_test_wine(&db, "To Promote", true);

        assert!(promote_wine(&db, &wine.short_id).unwrap());

        let promoted = get_wine_by_short_id(&db, &wine.short_id).unwrap().unwrap();
        assert!(!promoted.wishlist);

        assert_eq!(get_wishlist_wines(&db).unwrap().len(), 0);
        assert_eq!(get_cellar_wines(&db).unwrap().len(), 1);
    }

    #[test]
    fn promote_cellar_wine_returns_false() {
        let db = test_db();
        let wine = create_test_wine(&db, "Already Cellar", false);
        assert!(!promote_wine(&db, &wine.short_id).unwrap());
    }

    #[test]
    fn update_wine_works() {
        let db = test_db();
        let wine = create_test_wine(&db, "Old Name", false);

        let input = WineInput {
            name: "New Name",
            origin: "Italy",
            grape: "Sangiovese",
            notes: "Bold",
            sweetness: 4,
            acidity: 4,
            tannin: 4,
            alcohol: 4,
            body: 4,
            clarity: 4,
            color_intensity: 4,
            aroma_intensity: 4,
            nose_complexity: 4,
            background: "deep red",
        };
        let updated = update_wine(&db, &wine.short_id, &input).unwrap().unwrap();

        assert_eq!(updated.name, "New Name");
        assert_eq!(updated.origin, "Italy");
        assert_eq!(updated.sweetness, 4);
        assert_eq!(updated.background, "deep red");
    }

    #[test]
    fn update_wishlist_wine_works() {
        let db = test_db();
        let wine = create_test_wine(&db, "Wish", true);

        let updated = update_wishlist_wine(
            &db, &wine.short_id, "Updated Wish", "Spain", "Tempranillo", "Try soon", "amber",
        )
        .unwrap()
        .unwrap();
        assert_eq!(updated.name, "Updated Wish");
        assert!(updated.wishlist);
    }

    #[test]
    fn update_wishlist_wine_on_cellar_wine_returns_none() {
        let db = test_db();
        let wine = create_test_wine(&db, "Cellar", false);
        let result = update_wishlist_wine(&db, &wine.short_id, "X", "X", "X", "X", "").unwrap();
        assert!(result.is_none());
    }

    #[test]
    fn update_wine_image_and_get() {
        let db = test_db();
        let wine = create_test_wine(&db, "Photo Wine", false);
        assert!(!wine.has_image);

        let img_data = vec![0xFF, 0xD8, 0xFF]; // fake JPEG header
        assert!(update_wine_image(&db, &wine.short_id, &img_data, "image/jpeg").unwrap());

        let (data, mime) = get_wine_image(&db, &wine.short_id).unwrap().unwrap();
        assert_eq!(data, img_data);
        assert_eq!(mime, "image/jpeg");
    }

    #[test]
    fn get_wine_image_no_image() {
        let db = test_db();
        let wine = create_test_wine(&db, "No Photo", false);
        assert!(get_wine_image(&db, &wine.short_id).unwrap().is_none());
    }

    #[test]
    fn delete_wine_works() {
        let db = test_db();
        let wine = create_test_wine(&db, "Delete Me", false);
        assert!(delete_wine(&db, &wine.short_id).unwrap());
        assert!(get_wine_by_short_id(&db, &wine.short_id).unwrap().is_none());
    }

    #[test]
    fn delete_nonexistent_wine() {
        let db = test_db();
        assert!(!delete_wine(&db, "nope").unwrap());
    }

    // ── Sessions ───────────────────────────────────────────────────────

    #[test]
    fn session_lifecycle() {
        let db = test_db();
        insert_session(&db, "tok", "2099-01-01 00:00:00").unwrap();
        assert!(get_session_expiry(&db, "tok").unwrap().is_some());
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
}
