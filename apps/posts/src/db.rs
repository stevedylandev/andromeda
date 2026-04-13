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
pub struct Post {
    pub id: i64,
    pub short_id: String,
    pub title: String,
    pub slug: String,
    pub alias: Option<String>,
    pub canonical_url: Option<String>,
    pub published_date: Option<String>,
    pub meta_description: Option<String>,
    pub meta_image: Option<String>,
    pub lang: String,
    pub tags: Option<String>,
    pub content: String,
    pub status: String,
    pub created_at: String,
    pub updated_at: String,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct Page {
    pub id: i64,
    pub short_id: String,
    pub title: String,
    pub slug: String,
    pub content: String,
    pub is_published: bool,
    pub nav_order: i64,
    pub created_at: String,
    pub updated_at: String,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct UploadedFile {
    pub id: i64,
    pub short_id: String,
    pub filename: String,
    pub original_name: String,
    pub content_type: String,
    pub size: i64,
    pub created_at: String,
}

pub fn init_db() -> Db {
    let path = std::env::var("POSTS_DB_PATH").unwrap_or_else(|_| "posts.sqlite".to_string());
    let conn = Connection::open(&path).expect("Failed to open database");

    conn.execute_batch(
        "CREATE TABLE IF NOT EXISTS posts (
            id              INTEGER PRIMARY KEY AUTOINCREMENT,
            short_id        TEXT NOT NULL UNIQUE,
            title           TEXT NOT NULL,
            slug            TEXT NOT NULL UNIQUE,
            alias           TEXT,
            canonical_url   TEXT,
            published_date  TEXT,
            meta_description TEXT,
            meta_image      TEXT,
            lang            TEXT NOT NULL DEFAULT 'en',
            tags            TEXT,
            content         TEXT NOT NULL,
            status          TEXT NOT NULL DEFAULT 'draft',
            created_at      TEXT NOT NULL DEFAULT (datetime('now')),
            updated_at      TEXT NOT NULL DEFAULT (datetime('now'))
        );

        CREATE TABLE IF NOT EXISTS pages (
            id              INTEGER PRIMARY KEY AUTOINCREMENT,
            short_id        TEXT NOT NULL UNIQUE,
            title           TEXT NOT NULL,
            slug            TEXT NOT NULL UNIQUE,
            content         TEXT NOT NULL,
            is_published    INTEGER NOT NULL DEFAULT 0,
            nav_order       INTEGER NOT NULL DEFAULT 0,
            created_at      TEXT NOT NULL DEFAULT (datetime('now')),
            updated_at      TEXT NOT NULL DEFAULT (datetime('now'))
        );

        CREATE TABLE IF NOT EXISTS sessions (
            id              INTEGER PRIMARY KEY AUTOINCREMENT,
            token           TEXT NOT NULL UNIQUE,
            expires_at      TEXT NOT NULL
        );

        CREATE TABLE IF NOT EXISTS settings (
            key   TEXT PRIMARY KEY,
            value TEXT NOT NULL
        );

        CREATE TABLE IF NOT EXISTS files (
            id            INTEGER PRIMARY KEY AUTOINCREMENT,
            short_id      TEXT NOT NULL UNIQUE,
            filename      TEXT NOT NULL UNIQUE,
            original_name TEXT NOT NULL,
            content_type  TEXT NOT NULL DEFAULT 'application/octet-stream',
            size          INTEGER NOT NULL,
            created_at    TEXT NOT NULL DEFAULT (datetime('now'))
        );"
    )
    .expect("Failed to create tables");

    // Seed default settings
    conn.execute(
        "INSERT OR IGNORE INTO settings (key, value) VALUES ('blog_title', 'My Blog')",
        [],
    )
    .ok();
    conn.execute(
        "INSERT OR IGNORE INTO settings (key, value) VALUES ('blog_description', 'A simple blog')",
        [],
    )
    .ok();
    conn.execute(
        "INSERT OR IGNORE INTO settings (key, value) VALUES ('intro_content', '')",
        [],
    )
    .ok();
    conn.execute(
        "INSERT OR IGNORE INTO settings (key, value) VALUES ('nav_links', '[blog](/) [posts](/posts)')",
        [],
    )
    .ok();
    conn.execute(
        "INSERT OR IGNORE INTO settings (key, value) VALUES ('custom_css', '')",
        [],
    )
    .ok();
    conn.execute(
        "INSERT OR IGNORE INTO settings (key, value) VALUES ('favicon_url', '')",
        [],
    )
    .ok();
    conn.execute(
        "INSERT OR IGNORE INTO settings (key, value) VALUES ('og_image_url', '')",
        [],
    )
    .ok();
    conn.execute(
        "INSERT OR IGNORE INTO settings (key, value) VALUES ('custom_header', '')",
        [],
    )
    .ok();
    conn.execute(
        "INSERT OR IGNORE INTO settings (key, value) VALUES ('custom_footer', '<div>
<a href=\"/feed.xml\" class=\"rss-link\" title=\"RSS Feed\"><svg xmlns=\"http://www.w3.org/2000/svg\" width=\"32\" height=\"32\" fill=\"currentColor\" viewBox=\"0 0 256 256\"><path fill=\"currentColor\" d=\"M104.08 151.92A67.52 67.52 0 0 1 124 200a4 4 0 0 1-8 0a60 60 0 0 0-60-60a4 4 0 0 1 0-8a67.52 67.52 0 0 1 48.08 19.92M56 84a4 4 0 0 0 0 8a108 108 0 0 1 108 108a4 4 0 0 0 8 0A116 116 0 0 0 56 84m116 0A162.92 162.92 0 0 0 56 36a4 4 0 0 0 0 8a155 155 0 0 1 110.31 45.69A155 155 0 0 1 212 200a4 4 0 0 0 8 0a162.92 162.92 0 0 0-48-116M60 188a8 8 0 1 0 8 8a8 8 0 0 0-8-8\"/></svg></a>
</div>')",
        [],
    )
    .ok();

    Arc::new(Mutex::new(conn))
}

// --- Post CRUD ---

fn row_to_post(row: &rusqlite::Row) -> rusqlite::Result<Post> {
    Ok(Post {
        id: row.get(0)?,
        short_id: row.get(1)?,
        title: row.get(2)?,
        slug: row.get(3)?,
        alias: row.get(4)?,
        canonical_url: row.get(5)?,
        published_date: row.get(6)?,
        meta_description: row.get(7)?,
        meta_image: row.get(8)?,
        lang: row.get(9)?,
        tags: row.get(10)?,
        content: row.get(11)?,
        status: row.get(12)?,
        created_at: row.get(13)?,
        updated_at: row.get(14)?,
    })
}

const POST_COLS: &str = "id, short_id, title, slug, alias, canonical_url, published_date, meta_description, meta_image, lang, tags, content, status, created_at, updated_at";

pub fn create_post(
    db: &Db,
    title: &str,
    slug: &str,
    content: &str,
    status: &str,
    alias: Option<&str>,
    canonical_url: Option<&str>,
    published_date: Option<&str>,
    meta_description: Option<&str>,
    meta_image: Option<&str>,
    lang: &str,
    tags: Option<&str>,
) -> Result<Post, DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    let short_id = nanoid!(10);
    conn.execute(
        "INSERT INTO posts (short_id, title, slug, content, status, alias, canonical_url, published_date, meta_description, meta_image, lang, tags)
         VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7, ?8, ?9, ?10, ?11, ?12)",
        params![short_id, title, slug, content, status, alias, canonical_url, published_date, meta_description, meta_image, lang, tags],
    )?;
    let id = conn.last_insert_rowid();
    let post = conn.query_row(
        &format!("SELECT {} FROM posts WHERE id = ?1", POST_COLS),
        params![id],
        row_to_post,
    )?;
    Ok(post)
}

pub fn get_post_by_short_id(db: &Db, short_id: &str) -> Result<Option<Post>, DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    match conn.query_row(
        &format!("SELECT {} FROM posts WHERE short_id = ?1", POST_COLS),
        params![short_id],
        row_to_post,
    ) {
        Ok(post) => Ok(Some(post)),
        Err(rusqlite::Error::QueryReturnedNoRows) => Ok(None),
        Err(e) => Err(DbError::Sqlite(e)),
    }
}

pub fn get_post_by_slug(db: &Db, slug: &str) -> Result<Option<Post>, DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    match conn.query_row(
        &format!("SELECT {} FROM posts WHERE slug = ?1", POST_COLS),
        params![slug],
        row_to_post,
    ) {
        Ok(post) => Ok(Some(post)),
        Err(rusqlite::Error::QueryReturnedNoRows) => Ok(None),
        Err(e) => Err(DbError::Sqlite(e)),
    }
}

pub fn get_all_posts(db: &Db) -> Result<Vec<Post>, DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    let mut stmt = conn.prepare(
        &format!("SELECT {} FROM posts ORDER BY id DESC", POST_COLS),
    )?;
    let posts = stmt
        .query_map([], row_to_post)?
        .collect::<Result<Vec<_>, _>>()?;
    Ok(posts)
}

pub fn get_published_posts(db: &Db) -> Result<Vec<Post>, DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    let mut stmt = conn.prepare(
        &format!("SELECT {} FROM posts WHERE status = 'published' ORDER BY published_date DESC, id DESC", POST_COLS),
    )?;
    let posts = stmt
        .query_map([], row_to_post)?
        .collect::<Result<Vec<_>, _>>()?;
    Ok(posts)
}

pub fn update_post(
    db: &Db,
    short_id: &str,
    title: &str,
    slug: &str,
    content: &str,
    status: &str,
    alias: Option<&str>,
    canonical_url: Option<&str>,
    published_date: Option<&str>,
    meta_description: Option<&str>,
    meta_image: Option<&str>,
    lang: &str,
    tags: Option<&str>,
) -> Result<Option<Post>, DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    let effective_published_date = if status == "published" {
        Some(published_date.unwrap_or(""))
    } else {
        published_date
    };
    let rows = conn.execute(
        "UPDATE posts SET title = ?1, slug = ?2, content = ?3, status = ?4, alias = ?5, canonical_url = ?6,
         published_date = CASE WHEN ?4 = 'published' THEN COALESCE(?7, published_date, datetime('now')) ELSE ?7 END,
         meta_description = ?8, meta_image = ?9, lang = ?10, tags = ?11,
         updated_at = datetime('now') WHERE short_id = ?12",
        params![title, slug, content, status, alias, canonical_url, effective_published_date, meta_description, meta_image, lang, tags, short_id],
    )?;
    if rows == 0 {
        return Ok(None);
    }
    match conn.query_row(
        &format!("SELECT {} FROM posts WHERE short_id = ?1", POST_COLS),
        params![short_id],
        row_to_post,
    ) {
        Ok(post) => Ok(Some(post)),
        Err(rusqlite::Error::QueryReturnedNoRows) => Ok(None),
        Err(e) => Err(DbError::Sqlite(e)),
    }
}

pub fn delete_post(db: &Db, short_id: &str) -> Result<bool, DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    let rows = conn.execute("DELETE FROM posts WHERE short_id = ?1", params![short_id])?;
    Ok(rows > 0)
}

pub fn toggle_post_status(db: &Db, short_id: &str) -> Result<Option<String>, DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    let current: String = match conn.query_row(
        "SELECT status FROM posts WHERE short_id = ?1",
        params![short_id],
        |row| row.get(0),
    ) {
        Ok(s) => s,
        Err(rusqlite::Error::QueryReturnedNoRows) => return Ok(None),
        Err(e) => return Err(DbError::Sqlite(e)),
    };
    let new_status = if current == "published" { "draft" } else { "published" };
    if new_status == "published" {
        conn.execute(
            "UPDATE posts SET status = ?1, published_date = COALESCE(published_date, datetime('now')), updated_at = datetime('now') WHERE short_id = ?2",
            params![new_status, short_id],
        )?;
    } else {
        conn.execute(
            "UPDATE posts SET status = ?1, updated_at = datetime('now') WHERE short_id = ?2",
            params![new_status, short_id],
        )?;
    }
    Ok(Some(new_status.to_string()))
}

pub fn find_alias_redirect(db: &Db, alias: &str) -> Result<Option<String>, DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    match conn.query_row(
        "SELECT slug FROM posts WHERE alias = ?1 AND status = 'published'",
        params![alias],
        |row| row.get::<_, String>(0),
    ) {
        Ok(slug) => Ok(Some(format!("/posts/{}", slug))),
        Err(rusqlite::Error::QueryReturnedNoRows) => Ok(None),
        Err(e) => Err(DbError::Sqlite(e)),
    }
}

// --- Page CRUD ---

fn row_to_page(row: &rusqlite::Row) -> rusqlite::Result<Page> {
    Ok(Page {
        id: row.get(0)?,
        short_id: row.get(1)?,
        title: row.get(2)?,
        slug: row.get(3)?,
        content: row.get(4)?,
        is_published: row.get::<_, i64>(5)? != 0,
        nav_order: row.get(6)?,
        created_at: row.get(7)?,
        updated_at: row.get(8)?,
    })
}

const PAGE_COLS: &str = "id, short_id, title, slug, content, is_published, nav_order, created_at, updated_at";

pub fn create_page(
    db: &Db,
    title: &str,
    slug: &str,
    content: &str,
    is_published: bool,
    nav_order: i64,
) -> Result<Page, DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    let short_id = nanoid!(10);
    conn.execute(
        "INSERT INTO pages (short_id, title, slug, content, is_published, nav_order) VALUES (?1, ?2, ?3, ?4, ?5, ?6)",
        params![short_id, title, slug, content, is_published as i64, nav_order],
    )?;
    let id = conn.last_insert_rowid();
    let page = conn.query_row(
        &format!("SELECT {} FROM pages WHERE id = ?1", PAGE_COLS),
        params![id],
        row_to_page,
    )?;
    Ok(page)
}

pub fn get_page_by_short_id(db: &Db, short_id: &str) -> Result<Option<Page>, DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    match conn.query_row(
        &format!("SELECT {} FROM pages WHERE short_id = ?1", PAGE_COLS),
        params![short_id],
        row_to_page,
    ) {
        Ok(page) => Ok(Some(page)),
        Err(rusqlite::Error::QueryReturnedNoRows) => Ok(None),
        Err(e) => Err(DbError::Sqlite(e)),
    }
}

pub fn get_page_by_slug(db: &Db, slug: &str) -> Result<Option<Page>, DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    match conn.query_row(
        &format!("SELECT {} FROM pages WHERE slug = ?1", PAGE_COLS),
        params![slug],
        row_to_page,
    ) {
        Ok(page) => Ok(Some(page)),
        Err(rusqlite::Error::QueryReturnedNoRows) => Ok(None),
        Err(e) => Err(DbError::Sqlite(e)),
    }
}

pub fn get_all_pages(db: &Db) -> Result<Vec<Page>, DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    let mut stmt = conn.prepare(
        &format!("SELECT {} FROM pages ORDER BY nav_order ASC, id ASC", PAGE_COLS),
    )?;
    let pages = stmt
        .query_map([], row_to_page)?
        .collect::<Result<Vec<_>, _>>()?;
    Ok(pages)
}

pub fn get_published_pages(db: &Db) -> Result<Vec<Page>, DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    let mut stmt = conn.prepare(
        &format!("SELECT {} FROM pages WHERE is_published = 1 ORDER BY nav_order ASC, id ASC", PAGE_COLS),
    )?;
    let pages = stmt
        .query_map([], row_to_page)?
        .collect::<Result<Vec<_>, _>>()?;
    Ok(pages)
}

pub fn update_page(
    db: &Db,
    short_id: &str,
    title: &str,
    slug: &str,
    content: &str,
    is_published: bool,
    nav_order: i64,
) -> Result<Option<Page>, DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    let rows = conn.execute(
        "UPDATE pages SET title = ?1, slug = ?2, content = ?3, is_published = ?4, nav_order = ?5, updated_at = datetime('now') WHERE short_id = ?6",
        params![title, slug, content, is_published as i64, nav_order, short_id],
    )?;
    if rows == 0 {
        return Ok(None);
    }
    match conn.query_row(
        &format!("SELECT {} FROM pages WHERE short_id = ?1", PAGE_COLS),
        params![short_id],
        row_to_page,
    ) {
        Ok(page) => Ok(Some(page)),
        Err(rusqlite::Error::QueryReturnedNoRows) => Ok(None),
        Err(e) => Err(DbError::Sqlite(e)),
    }
}

pub fn delete_page(db: &Db, short_id: &str) -> Result<bool, DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    let rows = conn.execute("DELETE FROM pages WHERE short_id = ?1", params![short_id])?;
    Ok(rows > 0)
}

// --- Settings ---

pub fn get_setting(db: &Db, key: &str) -> Result<Option<String>, DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    match conn.query_row(
        "SELECT value FROM settings WHERE key = ?1",
        params![key],
        |row| row.get(0),
    ) {
        Ok(val) => Ok(Some(val)),
        Err(rusqlite::Error::QueryReturnedNoRows) => Ok(None),
        Err(e) => Err(DbError::Sqlite(e)),
    }
}

pub fn set_setting(db: &Db, key: &str, value: &str) -> Result<(), DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    conn.execute(
        "INSERT INTO settings (key, value) VALUES (?1, ?2) ON CONFLICT(key) DO UPDATE SET value = ?2",
        params![key, value],
    )?;
    Ok(())
}

pub fn get_all_settings(db: &Db) -> Result<Vec<(String, String)>, DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    let mut stmt = conn.prepare("SELECT key, value FROM settings ORDER BY key")?;
    let settings = stmt
        .query_map([], |row| Ok((row.get(0)?, row.get(1)?)))?
        .collect::<Result<Vec<_>, _>>()?;
    Ok(settings)
}

// --- Session functions ---

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

// --- File CRUD ---

fn row_to_file(row: &rusqlite::Row) -> rusqlite::Result<UploadedFile> {
    Ok(UploadedFile {
        id: row.get(0)?,
        short_id: row.get(1)?,
        filename: row.get(2)?,
        original_name: row.get(3)?,
        content_type: row.get(4)?,
        size: row.get(5)?,
        created_at: row.get(6)?,
    })
}

const FILE_COLS: &str = "id, short_id, filename, original_name, content_type, size, created_at";

pub fn create_file(
    db: &Db,
    filename: &str,
    original_name: &str,
    content_type: &str,
    size: i64,
) -> Result<UploadedFile, DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    let short_id = nanoid!(10);
    conn.execute(
        "INSERT INTO files (short_id, filename, original_name, content_type, size) VALUES (?1, ?2, ?3, ?4, ?5)",
        params![short_id, filename, original_name, content_type, size],
    )?;
    let id = conn.last_insert_rowid();
    let file = conn.query_row(
        &format!("SELECT {} FROM files WHERE id = ?1", FILE_COLS),
        params![id],
        row_to_file,
    )?;
    Ok(file)
}

pub fn get_all_files(db: &Db) -> Result<Vec<UploadedFile>, DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    let mut stmt = conn.prepare(
        &format!("SELECT {} FROM files ORDER BY id DESC", FILE_COLS),
    )?;
    let files = stmt
        .query_map([], row_to_file)?
        .collect::<Result<Vec<_>, _>>()?;
    Ok(files)
}

pub fn delete_file(db: &Db, short_id: &str) -> Result<Option<UploadedFile>, DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    let file = match conn.query_row(
        &format!("SELECT {} FROM files WHERE short_id = ?1", FILE_COLS),
        params![short_id],
        row_to_file,
    ) {
        Ok(f) => f,
        Err(rusqlite::Error::QueryReturnedNoRows) => return Ok(None),
        Err(e) => return Err(DbError::Sqlite(e)),
    };
    conn.execute("DELETE FROM files WHERE short_id = ?1", params![short_id])?;
    Ok(Some(file))
}

#[cfg(test)]
mod tests {
    use super::*;

    fn test_db() -> Db {
        let conn = Connection::open_in_memory().unwrap();
        conn.execute_batch(
            "CREATE TABLE IF NOT EXISTS posts (
                id              INTEGER PRIMARY KEY AUTOINCREMENT,
                short_id        TEXT NOT NULL UNIQUE,
                title           TEXT NOT NULL,
                slug            TEXT NOT NULL UNIQUE,
                alias           TEXT,
                canonical_url   TEXT,
                published_date  TEXT,
                meta_description TEXT,
                meta_image      TEXT,
                lang            TEXT NOT NULL DEFAULT 'en',
                tags            TEXT,
                content         TEXT NOT NULL,
                status          TEXT NOT NULL DEFAULT 'draft',
                created_at      TEXT NOT NULL DEFAULT (datetime('now')),
                updated_at      TEXT NOT NULL DEFAULT (datetime('now'))
            );
            CREATE TABLE IF NOT EXISTS pages (
                id              INTEGER PRIMARY KEY AUTOINCREMENT,
                short_id        TEXT NOT NULL UNIQUE,
                title           TEXT NOT NULL,
                slug            TEXT NOT NULL UNIQUE,
                content         TEXT NOT NULL,
                is_published    INTEGER NOT NULL DEFAULT 0,
                nav_order       INTEGER NOT NULL DEFAULT 0,
                created_at      TEXT NOT NULL DEFAULT (datetime('now')),
                updated_at      TEXT NOT NULL DEFAULT (datetime('now'))
            );
            CREATE TABLE IF NOT EXISTS sessions (
                id              INTEGER PRIMARY KEY AUTOINCREMENT,
                token           TEXT NOT NULL UNIQUE,
                expires_at      TEXT NOT NULL
            );
            CREATE TABLE IF NOT EXISTS settings (
                key   TEXT PRIMARY KEY,
                value TEXT NOT NULL
            );
            CREATE TABLE IF NOT EXISTS files (
                id            INTEGER PRIMARY KEY AUTOINCREMENT,
                short_id      TEXT NOT NULL UNIQUE,
                filename      TEXT NOT NULL UNIQUE,
                original_name TEXT NOT NULL,
                content_type  TEXT NOT NULL DEFAULT 'application/octet-stream',
                size          INTEGER NOT NULL,
                created_at    TEXT NOT NULL DEFAULT (datetime('now'))
            );",
        )
        .unwrap();
        Arc::new(Mutex::new(conn))
    }

    // ── Post CRUD ──────────────────────────────────────────────────────

    #[test]
    fn create_and_get_post() {
        let db = test_db();
        let post = create_post(
            &db, "Hello World", "hello-world", "# Hello", "draft",
            None, None, None, None, None, "en", None,
        )
        .unwrap();
        assert_eq!(post.title, "Hello World");
        assert_eq!(post.slug, "hello-world");
        assert_eq!(post.status, "draft");

        let fetched = get_post_by_short_id(&db, &post.short_id).unwrap().unwrap();
        assert_eq!(fetched.title, "Hello World");
    }

    #[test]
    fn get_post_by_slug_works() {
        let db = test_db();
        create_post(
            &db, "Test", "test-slug", "content", "published",
            None, None, Some("2024-01-01"), None, None, "en", None,
        )
        .unwrap();

        let post = get_post_by_slug(&db, "test-slug").unwrap().unwrap();
        assert_eq!(post.title, "Test");
    }

    #[test]
    fn duplicate_slug_fails() {
        let db = test_db();
        create_post(&db, "A", "same-slug", "a", "draft", None, None, None, None, None, "en", None).unwrap();
        let result = create_post(&db, "B", "same-slug", "b", "draft", None, None, None, None, None, "en", None);
        assert!(result.is_err());
    }

    #[test]
    fn get_all_posts_ordered_desc() {
        let db = test_db();
        create_post(&db, "First", "first", "a", "draft", None, None, None, None, None, "en", None).unwrap();
        create_post(&db, "Second", "second", "b", "draft", None, None, None, None, None, "en", None).unwrap();

        let all = get_all_posts(&db).unwrap();
        assert_eq!(all.len(), 2);
        assert_eq!(all[0].title, "Second");
        assert_eq!(all[1].title, "First");
    }

    #[test]
    fn get_published_posts_filters() {
        let db = test_db();
        create_post(&db, "Draft", "draft", "a", "draft", None, None, None, None, None, "en", None).unwrap();
        create_post(&db, "Published", "pub", "b", "published", None, None, Some("2024-01-01"), None, None, "en", None).unwrap();

        let published = get_published_posts(&db).unwrap();
        assert_eq!(published.len(), 1);
        assert_eq!(published[0].title, "Published");
    }

    #[test]
    fn delete_post_works() {
        let db = test_db();
        let post = create_post(&db, "Del", "del", "x", "draft", None, None, None, None, None, "en", None).unwrap();
        assert!(delete_post(&db, &post.short_id).unwrap());
        assert!(get_post_by_short_id(&db, &post.short_id).unwrap().is_none());
    }

    #[test]
    fn toggle_post_status_draft_to_published() {
        let db = test_db();
        let post = create_post(&db, "Toggle", "toggle", "x", "draft", None, None, None, None, None, "en", None).unwrap();
        let new_status = toggle_post_status(&db, &post.short_id).unwrap().unwrap();
        assert_eq!(new_status, "published");

        let updated = get_post_by_short_id(&db, &post.short_id).unwrap().unwrap();
        assert_eq!(updated.status, "published");
        assert!(updated.published_date.is_some());
    }

    #[test]
    fn toggle_post_status_published_to_draft() {
        let db = test_db();
        let post = create_post(&db, "Toggle", "toggle", "x", "published", None, None, Some("2024-01-01"), None, None, "en", None).unwrap();
        let new_status = toggle_post_status(&db, &post.short_id).unwrap().unwrap();
        assert_eq!(new_status, "draft");
    }

    #[test]
    fn find_alias_redirect_found() {
        let db = test_db();
        create_post(&db, "Aliased", "aliased-post", "x", "published", Some("old-url"), None, Some("2024-01-01"), None, None, "en", None).unwrap();
        let redirect = find_alias_redirect(&db, "old-url").unwrap();
        assert_eq!(redirect, Some("/posts/aliased-post".to_string()));
    }

    #[test]
    fn find_alias_redirect_not_found() {
        let db = test_db();
        assert!(find_alias_redirect(&db, "nonexistent").unwrap().is_none());
    }

    #[test]
    fn find_alias_redirect_only_published() {
        let db = test_db();
        create_post(&db, "Draft Alias", "draft-alias", "x", "draft", Some("my-alias"), None, None, None, None, "en", None).unwrap();
        assert!(find_alias_redirect(&db, "my-alias").unwrap().is_none());
    }

    // ── Page CRUD ──────────────────────────────────────────────────────

    #[test]
    fn create_and_get_page() {
        let db = test_db();
        let page = create_page(&db, "About", "about", "About content", true, 1).unwrap();
        assert_eq!(page.title, "About");
        assert!(page.is_published);

        let fetched = get_page_by_short_id(&db, &page.short_id).unwrap().unwrap();
        assert_eq!(fetched.slug, "about");
    }

    #[test]
    fn get_page_by_slug_works() {
        let db = test_db();
        create_page(&db, "Contact", "contact", "Email us", false, 2).unwrap();
        let page = get_page_by_slug(&db, "contact").unwrap().unwrap();
        assert_eq!(page.title, "Contact");
    }

    #[test]
    fn get_published_pages_filters() {
        let db = test_db();
        create_page(&db, "Pub", "pub", "x", true, 1).unwrap();
        create_page(&db, "Draft", "draft", "x", false, 2).unwrap();

        let published = get_published_pages(&db).unwrap();
        assert_eq!(published.len(), 1);
        assert_eq!(published[0].title, "Pub");
    }

    #[test]
    fn update_page_works() {
        let db = test_db();
        let page = create_page(&db, "Old", "old", "old content", false, 0).unwrap();
        let updated = update_page(&db, &page.short_id, "New", "new", "new content", true, 5)
            .unwrap()
            .unwrap();
        assert_eq!(updated.title, "New");
        assert!(updated.is_published);
        assert_eq!(updated.nav_order, 5);
    }

    #[test]
    fn delete_page_works() {
        let db = test_db();
        let page = create_page(&db, "Del", "del", "x", false, 0).unwrap();
        assert!(delete_page(&db, &page.short_id).unwrap());
        assert!(get_page_by_short_id(&db, &page.short_id).unwrap().is_none());
    }

    // ── Settings ───────────────────────────────────────────────────────

    #[test]
    fn settings_get_set() {
        let db = test_db();
        set_setting(&db, "blog_title", "My Blog").unwrap();
        let val = get_setting(&db, "blog_title").unwrap();
        assert_eq!(val, Some("My Blog".to_string()));
    }

    #[test]
    fn settings_upsert() {
        let db = test_db();
        set_setting(&db, "key", "first").unwrap();
        set_setting(&db, "key", "second").unwrap();
        assert_eq!(get_setting(&db, "key").unwrap(), Some("second".to_string()));
    }

    #[test]
    fn settings_missing_key() {
        let db = test_db();
        assert!(get_setting(&db, "nonexistent").unwrap().is_none());
    }

    #[test]
    fn get_all_settings_works() {
        let db = test_db();
        set_setting(&db, "a", "1").unwrap();
        set_setting(&db, "b", "2").unwrap();

        let all = get_all_settings(&db).unwrap();
        assert_eq!(all.len(), 2);
        assert_eq!(all[0], ("a".to_string(), "1".to_string()));
    }

    // ── File CRUD ──────────────────────────────────────────────────────

    #[test]
    fn create_and_get_files() {
        let db = test_db();
        let file = create_file(&db, "abc123.jpg", "photo.jpg", "image/jpeg", 1024).unwrap();
        assert_eq!(file.filename, "abc123.jpg");
        assert_eq!(file.original_name, "photo.jpg");
        assert_eq!(file.size, 1024);

        let all = get_all_files(&db).unwrap();
        assert_eq!(all.len(), 1);
    }

    #[test]
    fn delete_file_returns_deleted() {
        let db = test_db();
        let file = create_file(&db, "f.txt", "f.txt", "text/plain", 10).unwrap();
        let deleted = delete_file(&db, &file.short_id).unwrap();
        assert!(deleted.is_some());
        assert_eq!(deleted.unwrap().filename, "f.txt");

        assert!(get_all_files(&db).unwrap().is_empty());
    }

    #[test]
    fn delete_file_not_found() {
        let db = test_db();
        assert!(delete_file(&db, "nonexistent").unwrap().is_none());
    }

    // ── Sessions ───────────────────────────────────────────────────────

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
}
