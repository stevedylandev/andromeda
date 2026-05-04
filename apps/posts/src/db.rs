use nanoid::nanoid;
use rusqlite::{Connection, OptionalExtension, params};
use serde::{Deserialize, Serialize};
use std::sync::{Arc, Mutex};

pub use andromeda_db::{Db, DbError};
pub use andromeda_db::session::{insert_session, get_session_expiry, delete_session, prune_expired_sessions};

fn from_row<T: serde::de::DeserializeOwned>(row: &rusqlite::Row) -> rusqlite::Result<T> {
    serde_rusqlite::from_row::<T>(row).map_err(|e| {
        rusqlite::Error::FromSqlConversionFailure(0, rusqlite::types::Type::Null, Box::new(e))
    })
}

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct Post {
    pub id: i64,
    pub short_id: String,
    pub title: Option<String>,
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

impl Post {
    pub fn display_title(&self) -> String {
        if let Some(t) = self.title.as_ref().map(|s| s.trim()).filter(|s| !s.is_empty()) {
            return t.to_string();
        }
        let snippet: String = self
            .content
            .chars()
            .filter(|c| !matches!(c, '\n' | '\r'))
            .take(25)
            .collect();
        let snippet = snippet.trim();
        if snippet.is_empty() {
            "Untitled".to_string()
        } else if self.content.chars().count() > 60 {
            format!("{}…", snippet)
        } else {
            snippet.to_string()
        }
    }
}

#[derive(Serialize)]
pub struct PostInput<'a> {
    pub title: Option<&'a str>,
    pub slug: &'a str,
    pub content: &'a str,
    pub status: &'a str,
    pub alias: Option<&'a str>,
    pub canonical_url: Option<&'a str>,
    pub published_date: Option<&'a str>,
    pub meta_description: Option<&'a str>,
    pub meta_image: Option<&'a str>,
    pub lang: &'a str,
    pub tags: Option<&'a str>,
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
    pub storage_backend: String,
}

const SCHEMA: &str = "
    CREATE TABLE IF NOT EXISTS posts (
        id              INTEGER PRIMARY KEY AUTOINCREMENT,
        short_id        TEXT NOT NULL UNIQUE,
        title           TEXT,
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
        id              INTEGER PRIMARY KEY AUTOINCREMENT,
        short_id        TEXT NOT NULL UNIQUE,
        filename        TEXT NOT NULL UNIQUE,
        original_name   TEXT NOT NULL,
        content_type    TEXT NOT NULL DEFAULT 'application/octet-stream',
        size            INTEGER NOT NULL,
        created_at      TEXT NOT NULL DEFAULT (datetime('now')),
        storage_backend TEXT NOT NULL DEFAULT 'local'
    );
";

const DEFAULT_SETTINGS: &[(&str, &str)] = &[
    ("blog_title", "My Blog"),
    ("blog_description", "A simple blog"),
    ("intro_content", ""),
    ("nav_links", "[blog](/) [posts](/posts)"),
    ("custom_css", ""),
    ("favicon_url", ""),
    ("og_image_url", ""),
    ("custom_header", ""),
    ("custom_footer", "<div>
<a href=\"/feed.xml\" class=\"rss-link\" title=\"RSS Feed\"><svg xmlns=\"http://www.w3.org/2000/svg\" width=\"32\" height=\"32\" fill=\"currentColor\" viewBox=\"0 0 256 256\"><path fill=\"currentColor\" d=\"M104.08 151.92A67.52 67.52 0 0 1 124 200a4 4 0 0 1-8 0a60 60 0 0 0-60-60a4 4 0 0 1 0-8a67.52 67.52 0 0 1 48.08 19.92M56 84a4 4 0 0 0 0 8a108 108 0 0 1 108 108a4 4 0 0 0 8 0A116 116 0 0 0 56 84m116 0A162.92 162.92 0 0 0 56 36a4 4 0 0 0 0 8a155 155 0 0 1 110.31 45.69A155 155 0 0 1 212 200a4 4 0 0 0 8 0a162.92 162.92 0 0 0-48-116M60 188a8 8 0 1 0 8 8a8 8 0 0 0-8-8\"/></svg></a>
</div>"),
];

pub fn init_db() -> Db {
    let path = std::env::var("POSTS_DB_PATH").unwrap_or_else(|_| "posts.sqlite".to_string());
    let conn = Connection::open(&path).expect("Failed to open database");

    conn.execute_batch(SCHEMA).expect("Failed to create tables");
    migrate_post_title_nullable(&conn).expect("Failed to migrate posts.title");
    migrate_files_storage_backend(&conn).expect("Failed to migrate files.storage_backend");

    for (key, value) in DEFAULT_SETTINGS {
        conn.execute(
            "INSERT OR IGNORE INTO settings (key, value) VALUES (?1, ?2)",
            params![key, value],
        )
        .ok();
    }

    Arc::new(Mutex::new(conn))
}

fn migrate_post_title_nullable(conn: &Connection) -> rusqlite::Result<()> {
    let title_notnull: i64 = conn
        .query_row(
            "SELECT \"notnull\" FROM pragma_table_info('posts') WHERE name = 'title'",
            [],
            |row| row.get(0),
        )
        .unwrap_or(0);
    if title_notnull == 0 {
        return Ok(());
    }
    conn.execute_batch(
        "BEGIN;
         CREATE TABLE posts_new (
            id              INTEGER PRIMARY KEY AUTOINCREMENT,
            short_id        TEXT NOT NULL UNIQUE,
            title           TEXT,
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
         INSERT INTO posts_new (id, short_id, title, slug, alias, canonical_url, published_date, meta_description, meta_image, lang, tags, content, status, created_at, updated_at)
            SELECT id, short_id, title, slug, alias, canonical_url, published_date, meta_description, meta_image, lang, tags, content, status, created_at, updated_at FROM posts;
         DROP TABLE posts;
         ALTER TABLE posts_new RENAME TO posts;
         COMMIT;",
    )
}

fn migrate_files_storage_backend(conn: &Connection) -> rusqlite::Result<()> {
    let exists: i64 = conn
        .query_row(
            "SELECT COUNT(*) FROM pragma_table_info('files') WHERE name = 'storage_backend'",
            [],
            |row| row.get(0),
        )
        .unwrap_or(0);
    if exists > 0 {
        return Ok(());
    }
    conn.execute(
        "ALTER TABLE files ADD COLUMN storage_backend TEXT NOT NULL DEFAULT 'local'",
        [],
    )?;
    Ok(())
}

// --- Post CRUD ---

const POST_COLS: &str = "id, short_id, title, slug, alias, canonical_url, published_date, meta_description, meta_image, lang, tags, content, status, created_at, updated_at";

pub fn create_post(db: &Db, input: &PostInput) -> Result<Post, DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    let short_id = nanoid!(10);
    let named = serde_rusqlite::to_params_named(input)
        .map_err(|e| rusqlite::Error::ToSqlConversionFailure(Box::new(e)))?;
    let mut bindings = named.to_slice();
    bindings.push((":short_id", &short_id));
    conn.execute(
        "INSERT INTO posts (short_id, title, slug, content, status, alias, canonical_url, published_date, meta_description, meta_image, lang, tags)
         VALUES (:short_id, :title, :slug, :content, :status, :alias, :canonical_url, :published_date, :meta_description, :meta_image, :lang, :tags)",
        bindings.as_slice(),
    )?;
    let id = conn.last_insert_rowid();
    let post = conn.query_row(
        &format!("SELECT {} FROM posts WHERE id = ?1", POST_COLS),
        params![id],
        from_row,
    )?;
    Ok(post)
}

pub fn get_post_by_short_id(db: &Db, short_id: &str) -> Result<Option<Post>, DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    let post = conn
        .query_row(
            &format!("SELECT {} FROM posts WHERE short_id = ?1", POST_COLS),
            params![short_id],
            from_row,
        )
        .optional()?;
    Ok(post)
}

pub fn get_post_by_slug(db: &Db, slug: &str) -> Result<Option<Post>, DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    let post = conn
        .query_row(
            &format!("SELECT {} FROM posts WHERE slug = ?1", POST_COLS),
            params![slug],
            from_row,
        )
        .optional()?;
    Ok(post)
}

pub fn get_all_posts(db: &Db) -> Result<Vec<Post>, DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    let mut stmt = conn.prepare(
        &format!("SELECT {} FROM posts ORDER BY id DESC", POST_COLS),
    )?;
    let posts = stmt
        .query_map([], from_row)?
        .collect::<Result<Vec<_>, _>>()?;
    Ok(posts)
}

pub fn get_published_posts(db: &Db, limit: Option<i64>) -> Result<Vec<Post>, DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    let limit_value = limit.unwrap_or(-1);
    let mut stmt = conn.prepare(
        &format!("SELECT {} FROM posts WHERE status = 'published' ORDER BY published_date DESC, id DESC LIMIT ?1", POST_COLS),
    )?;
    let posts = stmt
        .query_map(params![limit_value], from_row)?
        .collect::<Result<Vec<_>, _>>()?;
    Ok(posts)
}

pub fn update_post(db: &Db, short_id: &str, input: &PostInput) -> Result<Option<Post>, DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    let effective_published_date = if input.status == "published" {
        Some(input.published_date.unwrap_or(""))
    } else {
        input.published_date
    };
    let rows = conn.execute(
        "UPDATE posts SET title = ?1, slug = ?2, content = ?3, status = ?4, alias = ?5, canonical_url = ?6,
         published_date = CASE WHEN ?4 = 'published' THEN COALESCE(?7, published_date, datetime('now')) ELSE ?7 END,
         meta_description = ?8, meta_image = ?9, lang = ?10, tags = ?11,
         updated_at = datetime('now') WHERE short_id = ?12",
        params![input.title, input.slug, input.content, input.status, input.alias, input.canonical_url, effective_published_date, input.meta_description, input.meta_image, input.lang, input.tags, short_id],
    )?;
    if rows == 0 {
        return Ok(None);
    }
    let post = conn
        .query_row(
            &format!("SELECT {} FROM posts WHERE short_id = ?1", POST_COLS),
            params![short_id],
            from_row,
        )
        .optional()?;
    Ok(post)
}

pub fn delete_post(db: &Db, short_id: &str) -> Result<bool, DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    let rows = conn.execute("DELETE FROM posts WHERE short_id = ?1", params![short_id])?;
    Ok(rows > 0)
}

pub fn toggle_post_status(db: &Db, short_id: &str) -> Result<Option<String>, DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    let current: Option<String> = conn
        .query_row(
            "SELECT status FROM posts WHERE short_id = ?1",
            params![short_id],
            |row| row.get(0),
        )
        .optional()?;
    let current = match current {
        Some(s) => s,
        None => return Ok(None),
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
    let slug: Option<String> = conn
        .query_row(
            "SELECT slug FROM posts WHERE alias = ?1 AND status = 'published'",
            params![alias],
            |row| row.get(0),
        )
        .optional()?;
    Ok(slug.map(|s| format!("/posts/{}", s)))
}

// --- Page CRUD ---

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
        from_row,
    )?;
    Ok(page)
}

pub fn get_page_by_short_id(db: &Db, short_id: &str) -> Result<Option<Page>, DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    let page = conn
        .query_row(
            &format!("SELECT {} FROM pages WHERE short_id = ?1", PAGE_COLS),
            params![short_id],
            from_row,
        )
        .optional()?;
    Ok(page)
}

pub fn get_page_by_slug(db: &Db, slug: &str) -> Result<Option<Page>, DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    let page = conn
        .query_row(
            &format!("SELECT {} FROM pages WHERE slug = ?1", PAGE_COLS),
            params![slug],
            from_row,
        )
        .optional()?;
    Ok(page)
}

pub fn get_all_pages(db: &Db) -> Result<Vec<Page>, DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    let mut stmt = conn.prepare(
        &format!("SELECT {} FROM pages ORDER BY nav_order ASC, id ASC", PAGE_COLS),
    )?;
    let pages = stmt
        .query_map([], from_row)?
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
    let page = conn
        .query_row(
            &format!("SELECT {} FROM pages WHERE short_id = ?1", PAGE_COLS),
            params![short_id],
            from_row,
        )
        .optional()?;
    Ok(page)
}

pub fn delete_page(db: &Db, short_id: &str) -> Result<bool, DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    let rows = conn.execute("DELETE FROM pages WHERE short_id = ?1", params![short_id])?;
    Ok(rows > 0)
}

// --- Settings ---

pub fn get_setting(db: &Db, key: &str) -> Result<Option<String>, DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    let val = conn
        .query_row(
            "SELECT value FROM settings WHERE key = ?1",
            params![key],
            |row| row.get(0),
        )
        .optional()?;
    Ok(val)
}

pub fn set_setting(db: &Db, key: &str, value: &str) -> Result<(), DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    conn.execute(
        "INSERT INTO settings (key, value) VALUES (?1, ?2) ON CONFLICT(key) DO UPDATE SET value = ?2",
        params![key, value],
    )?;
    Ok(())
}

// --- File CRUD ---

const FILE_COLS: &str = "id, short_id, filename, original_name, content_type, size, created_at, storage_backend";

pub fn create_file(
    db: &Db,
    filename: &str,
    original_name: &str,
    content_type: &str,
    size: i64,
    storage_backend: &str,
) -> Result<UploadedFile, DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    let short_id = nanoid!(10);
    conn.execute(
        "INSERT INTO files (short_id, filename, original_name, content_type, size, storage_backend) VALUES (?1, ?2, ?3, ?4, ?5, ?6)",
        params![short_id, filename, original_name, content_type, size, storage_backend],
    )?;
    let id = conn.last_insert_rowid();
    let file = conn.query_row(
        &format!("SELECT {} FROM files WHERE id = ?1", FILE_COLS),
        params![id],
        from_row,
    )?;
    Ok(file)
}

pub fn get_file_by_filename(db: &Db, filename: &str) -> Result<Option<UploadedFile>, DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    let file = conn
        .query_row(
            &format!("SELECT {} FROM files WHERE filename = ?1", FILE_COLS),
            params![filename],
            from_row,
        )
        .optional()?;
    Ok(file)
}

pub fn get_all_files(db: &Db) -> Result<Vec<UploadedFile>, DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    let mut stmt = conn.prepare(
        &format!("SELECT {} FROM files ORDER BY id DESC", FILE_COLS),
    )?;
    let files = stmt
        .query_map([], from_row)?
        .collect::<Result<Vec<_>, _>>()?;
    Ok(files)
}

pub fn delete_file(db: &Db, short_id: &str) -> Result<Option<UploadedFile>, DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    let file: Option<UploadedFile> = conn
        .query_row(
            &format!("SELECT {} FROM files WHERE short_id = ?1", FILE_COLS),
            params![short_id],
            from_row,
        )
        .optional()?;
    match file {
        Some(f) => {
            conn.execute("DELETE FROM files WHERE short_id = ?1", params![short_id])?;
            Ok(Some(f))
        }
        None => Ok(None),
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn test_db() -> Db {
        let conn = Connection::open_in_memory().unwrap();
        conn.execute_batch(SCHEMA).unwrap();
        Arc::new(Mutex::new(conn))
    }

    fn test_post_input<'a>(title: &'a str, slug: &'a str, content: &'a str, status: &'a str) -> PostInput<'a> {
        PostInput {
            title: Some(title),
            slug,
            content,
            status,
            alias: None,
            canonical_url: None,
            published_date: None,
            meta_description: None,
            meta_image: None,
            lang: "en",
            tags: None,
        }
    }

    // ── Post CRUD ──────────────────────────────────────────────────────

    #[test]
    fn create_and_get_post() {
        let db = test_db();
        let post = create_post(&db, &test_post_input("Hello World", "hello-world", "# Hello", "draft")).unwrap();
        assert_eq!(post.title.as_deref(), Some("Hello World"));
        assert_eq!(post.slug, "hello-world");
        assert_eq!(post.status, "draft");

        let fetched = get_post_by_short_id(&db, &post.short_id).unwrap().unwrap();
        assert_eq!(fetched.title.as_deref(), Some("Hello World"));
    }

    #[test]
    fn create_post_without_title() {
        let db = test_db();
        let input = PostInput {
            title: None,
            slug: "no-title",
            content: "just a quick thought",
            status: "draft",
            alias: None,
            canonical_url: None,
            published_date: None,
            meta_description: None,
            meta_image: None,
            lang: "en",
            tags: None,
        };
        let post = create_post(&db, &input).unwrap();
        assert!(post.title.is_none());
        assert_eq!(post.display_title(), "just a quick thought");
    }

    #[test]
    fn get_post_by_slug_works() {
        let db = test_db();
        let mut input = test_post_input("Test", "test-slug", "content", "published");
        input.published_date = Some("2024-01-01");
        create_post(&db, &input).unwrap();

        let post = get_post_by_slug(&db, "test-slug").unwrap().unwrap();
        assert_eq!(post.title.as_deref(), Some("Test"));
    }

    #[test]
    fn duplicate_slug_fails() {
        let db = test_db();
        create_post(&db, &test_post_input("A", "same-slug", "a", "draft")).unwrap();
        let result = create_post(&db, &test_post_input("B", "same-slug", "b", "draft"));
        assert!(result.is_err());
    }

    #[test]
    fn get_all_posts_ordered_desc() {
        let db = test_db();
        create_post(&db, &test_post_input("First", "first", "a", "draft")).unwrap();
        create_post(&db, &test_post_input("Second", "second", "b", "draft")).unwrap();

        let all = get_all_posts(&db).unwrap();
        assert_eq!(all.len(), 2);
        assert_eq!(all[0].title.as_deref(), Some("Second"));
        assert_eq!(all[1].title.as_deref(), Some("First"));
    }

    #[test]
    fn get_published_posts_filters() {
        let db = test_db();
        create_post(&db, &test_post_input("Draft", "draft", "a", "draft")).unwrap();
        let mut input = test_post_input("Published", "pub", "b", "published");
        input.published_date = Some("2024-01-01");
        create_post(&db, &input).unwrap();

        let published = get_published_posts(&db, None).unwrap();
        assert_eq!(published.len(), 1);
        assert_eq!(published[0].title.as_deref(), Some("Published"));
    }

    #[test]
    fn delete_post_works() {
        let db = test_db();
        let post = create_post(&db, &test_post_input("Del", "del", "x", "draft")).unwrap();
        assert!(delete_post(&db, &post.short_id).unwrap());
        assert!(get_post_by_short_id(&db, &post.short_id).unwrap().is_none());
    }

    #[test]
    fn toggle_post_status_draft_to_published() {
        let db = test_db();
        let post = create_post(&db, &test_post_input("Toggle", "toggle", "x", "draft")).unwrap();
        let new_status = toggle_post_status(&db, &post.short_id).unwrap().unwrap();
        assert_eq!(new_status, "published");

        let updated = get_post_by_short_id(&db, &post.short_id).unwrap().unwrap();
        assert_eq!(updated.status, "published");
        assert!(updated.published_date.is_some());
    }

    #[test]
    fn toggle_post_status_published_to_draft() {
        let db = test_db();
        let mut input = test_post_input("Toggle", "toggle", "x", "published");
        input.published_date = Some("2024-01-01");
        let post = create_post(&db, &input).unwrap();
        let new_status = toggle_post_status(&db, &post.short_id).unwrap().unwrap();
        assert_eq!(new_status, "draft");
    }

    #[test]
    fn find_alias_redirect_found() {
        let db = test_db();
        let mut input = test_post_input("Aliased", "aliased-post", "x", "published");
        input.alias = Some("old-url");
        input.published_date = Some("2024-01-01");
        create_post(&db, &input).unwrap();
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
        let mut input = test_post_input("Draft Alias", "draft-alias", "x", "draft");
        input.alias = Some("my-alias");
        create_post(&db, &input).unwrap();
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

    // ── File CRUD ──────────────────────────────────────────────────────

    #[test]
    fn create_and_get_files() {
        let db = test_db();
        let file = create_file(&db, "abc123.jpg", "photo.jpg", "image/jpeg", 1024, "local").unwrap();
        assert_eq!(file.filename, "abc123.jpg");
        assert_eq!(file.original_name, "photo.jpg");
        assert_eq!(file.size, 1024);

        let all = get_all_files(&db).unwrap();
        assert_eq!(all.len(), 1);
    }

    #[test]
    fn delete_file_returns_deleted() {
        let db = test_db();
        let file = create_file(&db, "f.txt", "f.txt", "text/plain", 10, "local").unwrap();
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
