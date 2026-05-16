package main

import (
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/stevedylandev/andromeda/crates-go/auth"
	_ "modernc.org/sqlite"
)

const postsSchema = `
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
`

var defaultSettings = [][2]string{
	{"blog_title", "My Blog"},
	{"blog_description", "A simple blog"},
	{"intro_content", ""},
	{"nav_links", "[blog](/) [posts](/posts)"},
	{"custom_css", ""},
	{"favicon_url", ""},
	{"og_image_url", ""},
	{"custom_header", ""},
	{"custom_footer", `<div>
<a href="/feed.xml" class="rss-link" title="RSS Feed"><svg xmlns="http://www.w3.org/2000/svg" width="32" height="32" fill="currentColor" viewBox="0 0 256 256"><path fill="currentColor" d="M104.08 151.92A67.52 67.52 0 0 1 124 200a4 4 0 0 1-8 0a60 60 0 0 0-60-60a4 4 0 0 1 0-8a67.52 67.52 0 0 1 48.08 19.92M56 84a4 4 0 0 0 0 8a108 108 0 0 1 108 108a4 4 0 0 0 8 0A116 116 0 0 0 56 84m116 0A162.92 162.92 0 0 0 56 36a4 4 0 0 0 0 8a155 155 0 0 1 110.31 45.69A155 155 0 0 1 212 200a4 4 0 0 0 8 0a162.92 162.92 0 0 0-48-116M60 188a8 8 0 1 0 8 8a8 8 0 0 0-8-8"/></svg></a>
</div>`},
}

func openDB(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if _, err := db.Exec(postsSchema); err != nil {
		return nil, err
	}
	for _, kv := range defaultSettings {
		_, _ = db.Exec(`INSERT OR IGNORE INTO settings (key, value) VALUES (?, ?)`, kv[0], kv[1])
	}
	return db, nil
}

const postCols = `id, short_id, title, slug, alias, canonical_url, published_date, meta_description, meta_image, lang, tags, content, status, created_at, updated_at`

func scanPost(s interface{ Scan(...any) error }) (*Post, error) {
	var p Post
	var title, alias, canonicalURL, publishedDate, metaDesc, metaImage, tags sql.NullString
	err := s.Scan(&p.ID, &p.ShortID, &title, &p.Slug, &alias, &canonicalURL,
		&publishedDate, &metaDesc, &metaImage, &p.Lang, &tags, &p.Content,
		&p.Status, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if title.Valid {
		v := title.String
		p.Title = &v
	}
	if alias.Valid {
		v := alias.String
		p.Alias = &v
	}
	if canonicalURL.Valid {
		v := canonicalURL.String
		p.CanonicalURL = &v
	}
	if publishedDate.Valid {
		v := publishedDate.String
		p.PublishedDate = &v
	}
	if metaDesc.Valid {
		v := metaDesc.String
		p.MetaDescription = &v
	}
	if metaImage.Valid {
		v := metaImage.String
		p.MetaImage = &v
	}
	if tags.Valid {
		v := tags.String
		p.Tags = &v
	}
	return &p, nil
}

type PostInput struct {
	Title           *string
	Slug            string
	Content         string
	Status          string
	Alias           *string
	CanonicalURL    *string
	PublishedDate   *string
	MetaDescription *string
	MetaImage       *string
	Lang            string
	Tags            *string
}

func nullable(p *string) any {
	if p == nil {
		return nil
	}
	return *p
}

func createPost(db *sql.DB, in PostInput) (*Post, error) {
	shortID, err := auth.GenerateShortID(10)
	if err != nil {
		return nil, err
	}
	res, err := db.Exec(
		`INSERT INTO posts (short_id, title, slug, content, status, alias, canonical_url, published_date, meta_description, meta_image, lang, tags)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		shortID, nullable(in.Title), in.Slug, in.Content, in.Status,
		nullable(in.Alias), nullable(in.CanonicalURL), nullable(in.PublishedDate),
		nullable(in.MetaDescription), nullable(in.MetaImage), in.Lang, nullable(in.Tags),
	)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return scanPost(db.QueryRow(`SELECT `+postCols+` FROM posts WHERE id = ?`, id))
}

func getPostByShortID(db *sql.DB, shortID string) (*Post, error) {
	return scanPost(db.QueryRow(`SELECT `+postCols+` FROM posts WHERE short_id = ?`, shortID))
}

func getPostBySlug(db *sql.DB, slug string) (*Post, error) {
	return scanPost(db.QueryRow(`SELECT `+postCols+` FROM posts WHERE slug = ?`, slug))
}

func getAllPosts(db *sql.DB) ([]Post, error) {
	rows, err := db.Query(`SELECT ` + postCols + ` FROM posts ORDER BY id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Post
	for rows.Next() {
		p, err := scanPost(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	return out, rows.Err()
}

func getPublishedPosts(db *sql.DB, limit int64) ([]Post, error) {
	if limit <= 0 {
		limit = -1
	}
	rows, err := db.Query(
		`SELECT `+postCols+` FROM posts WHERE status = 'published' ORDER BY published_date DESC, id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Post
	for rows.Next() {
		p, err := scanPost(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	return out, rows.Err()
}

func updatePost(db *sql.DB, shortID string, in PostInput) (*Post, error) {
	res, err := db.Exec(
		`UPDATE posts SET title = ?, slug = ?, content = ?, status = ?, alias = ?, canonical_url = ?,
		 published_date = CASE WHEN ? = 'published' THEN COALESCE(?, published_date, datetime('now')) ELSE ? END,
		 meta_description = ?, meta_image = ?, lang = ?, tags = ?,
		 updated_at = datetime('now') WHERE short_id = ?`,
		nullable(in.Title), in.Slug, in.Content, in.Status, nullable(in.Alias), nullable(in.CanonicalURL),
		in.Status, nullable(in.PublishedDate), nullable(in.PublishedDate),
		nullable(in.MetaDescription), nullable(in.MetaImage), in.Lang, nullable(in.Tags), shortID,
	)
	if err != nil {
		return nil, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil, nil
	}
	return getPostByShortID(db, shortID)
}

func deletePost(db *sql.DB, shortID string) (bool, error) {
	res, err := db.Exec(`DELETE FROM posts WHERE short_id = ?`, shortID)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func togglePostStatus(db *sql.DB, shortID string) (string, error) {
	var current string
	err := db.QueryRow(`SELECT status FROM posts WHERE short_id = ?`, shortID).Scan(&current)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	newStatus := "published"
	if current == "published" {
		newStatus = "draft"
	}
	if newStatus == "published" {
		_, err = db.Exec(
			`UPDATE posts SET status = ?, published_date = COALESCE(published_date, datetime('now')), updated_at = datetime('now') WHERE short_id = ?`,
			newStatus, shortID)
	} else {
		_, err = db.Exec(`UPDATE posts SET status = ?, updated_at = datetime('now') WHERE short_id = ?`,
			newStatus, shortID)
	}
	return newStatus, err
}

func findAliasRedirect(db *sql.DB, alias string) (string, error) {
	var slug string
	err := db.QueryRow(`SELECT slug FROM posts WHERE alias = ? AND status = 'published'`, alias).Scan(&slug)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return "/posts/" + slug, nil
}

const pageCols = `id, short_id, title, slug, content, is_published, nav_order, created_at, updated_at`

func scanPage(s interface{ Scan(...any) error }) (*Page, error) {
	var p Page
	var pub int
	err := s.Scan(&p.ID, &p.ShortID, &p.Title, &p.Slug, &p.Content, &pub, &p.NavOrder, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	p.IsPublished = pub != 0
	return &p, nil
}

func createPage(db *sql.DB, title, slug, content string, isPublished bool, navOrder int64) (*Page, error) {
	shortID, err := auth.GenerateShortID(10)
	if err != nil {
		return nil, err
	}
	pub := 0
	if isPublished {
		pub = 1
	}
	res, err := db.Exec(
		`INSERT INTO pages (short_id, title, slug, content, is_published, nav_order) VALUES (?, ?, ?, ?, ?, ?)`,
		shortID, title, slug, content, pub, navOrder)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return scanPage(db.QueryRow(`SELECT `+pageCols+` FROM pages WHERE id = ?`, id))
}

func getPageByShortID(db *sql.DB, shortID string) (*Page, error) {
	return scanPage(db.QueryRow(`SELECT `+pageCols+` FROM pages WHERE short_id = ?`, shortID))
}

func getPageBySlug(db *sql.DB, slug string) (*Page, error) {
	return scanPage(db.QueryRow(`SELECT `+pageCols+` FROM pages WHERE slug = ?`, slug))
}

func getAllPages(db *sql.DB) ([]Page, error) {
	rows, err := db.Query(`SELECT ` + pageCols + ` FROM pages ORDER BY nav_order ASC, id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Page
	for rows.Next() {
		p, err := scanPage(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	return out, rows.Err()
}

func updatePage(db *sql.DB, shortID, title, slug, content string, isPublished bool, navOrder int64) (*Page, error) {
	pub := 0
	if isPublished {
		pub = 1
	}
	res, err := db.Exec(
		`UPDATE pages SET title = ?, slug = ?, content = ?, is_published = ?, nav_order = ?, updated_at = datetime('now') WHERE short_id = ?`,
		title, slug, content, pub, navOrder, shortID)
	if err != nil {
		return nil, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil, nil
	}
	return getPageByShortID(db, shortID)
}

func deletePage(db *sql.DB, shortID string) error {
	_, err := db.Exec(`DELETE FROM pages WHERE short_id = ?`, shortID)
	return err
}

func getSetting(db *sql.DB, key string) (string, error) {
	var v string
	err := db.QueryRow(`SELECT value FROM settings WHERE key = ?`, key).Scan(&v)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return v, nil
}

func setSetting(db *sql.DB, key, value string) error {
	_, err := db.Exec(`INSERT INTO settings (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
		key, value)
	return err
}

const fileCols = `id, short_id, filename, original_name, content_type, size, created_at, storage_backend`

func scanFile(s interface{ Scan(...any) error }) (*UploadedFile, error) {
	var f UploadedFile
	err := s.Scan(&f.ID, &f.ShortID, &f.Filename, &f.OriginalName, &f.ContentType, &f.Size, &f.CreatedAt, &f.StorageBackend)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func createFile(db *sql.DB, filename, originalName, contentType string, size int64) (*UploadedFile, error) {
	shortID, err := auth.GenerateShortID(10)
	if err != nil {
		return nil, err
	}
	res, err := db.Exec(
		`INSERT INTO files (short_id, filename, original_name, content_type, size, storage_backend) VALUES (?, ?, ?, ?, ?, 'local')`,
		shortID, filename, originalName, contentType, size)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return scanFile(db.QueryRow(`SELECT `+fileCols+` FROM files WHERE id = ?`, id))
}

func getFileByFilename(db *sql.DB, filename string) (*UploadedFile, error) {
	return scanFile(db.QueryRow(`SELECT `+fileCols+` FROM files WHERE filename = ?`, filename))
}

func getAllFiles(db *sql.DB) ([]UploadedFile, error) {
	rows, err := db.Query(`SELECT ` + fileCols + ` FROM files ORDER BY id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []UploadedFile
	for rows.Next() {
		f, err := scanFile(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *f)
	}
	return out, rows.Err()
}

func deleteFile(db *sql.DB, shortID string) (*UploadedFile, error) {
	f, err := scanFile(db.QueryRow(`SELECT `+fileCols+` FROM files WHERE short_id = ?`, shortID))
	if err != nil || f == nil {
		return f, err
	}
	if _, err := db.Exec(`DELETE FROM files WHERE short_id = ?`, shortID); err != nil {
		return nil, err
	}
	return f, nil
}

func nowDatetime() string {
	return time.Now().UTC().Format("2006-01-02 15:04:05")
}

func _useStrings() {
	_ = strings.TrimSpace
}
