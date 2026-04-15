use askama::Template;
use axum::{
    extract::DefaultBodyLimit,
    routing::{get, post},
    Router,
};
use pulldown_cmark::{Options, Parser, html};
use rust_embed::Embed;
use std::sync::Arc;

use crate::db::{self, Db, Page, Post, UploadedFile};

mod handlers;

#[derive(Debug, Clone)]
pub struct NavLink {
    pub label: String,
    pub url: String,
}

#[derive(Clone)]
pub struct AppState {
    pub db: Db,
    pub app_password: String,
    pub cookie_secure: bool,
    pub uploads_dir: String,
    pub site_url: String,
}

#[derive(Embed)]
#[folder = "static/"]
struct Static;

// --- Templates ---

#[derive(Template)]
#[template(path = "base.html")]
struct BaseTemplate {
    blog_title: String,
    nav_links: Vec<NavLink>,
    favicon_url: String,
    og_image_url: String,
    header_html: String,
    footer_html: String,
}

#[derive(Template)]
#[template(path = "admin_base.html")]
struct AdminBaseTemplate;

#[derive(Template)]
#[template(path = "login.html")]
struct LoginTemplate {
    error: Option<String>,
}

#[derive(Template)]
#[template(path = "index.html")]
struct IndexTemplate {
    blog_title: String,
    blog_description: String,
    intro_html: String,
    posts: Vec<Post>,
    nav_links: Vec<NavLink>,
    favicon_url: String,
    og_image_url: String,
    site_url: String,
    header_html: String,
    footer_html: String,
}

#[derive(Template)]
#[template(path = "post.html")]
struct PostTemplate {
    blog_title: String,
    nav_links: Vec<NavLink>,
    post: Post,
    rendered_content: String,
    favicon_url: String,
    og_image_url: String,
    site_url: String,
    header_html: String,
    footer_html: String,
}

#[derive(Template)]
#[template(path = "page.html")]
struct PageTemplate {
    blog_title: String,
    nav_links: Vec<NavLink>,
    page: Page,
    rendered_content: String,
    favicon_url: String,
    og_image_url: String,
    site_url: String,
    header_html: String,
    footer_html: String,
}

#[derive(Template)]
#[template(path = "admin_index.html")]
struct AdminIndexTemplate {
    posts: Vec<Post>,
}

#[derive(Template)]
#[template(path = "admin_post_form.html")]
struct AdminPostFormTemplate {
    post: Option<Post>,
    error: Option<String>,
}

#[derive(Template)]
#[template(path = "admin_pages.html")]
struct AdminPagesTemplate {
    pages: Vec<Page>,
}

#[derive(Template)]
#[template(path = "admin_page_form.html")]
struct AdminPageFormTemplate {
    page: Option<Page>,
    error: Option<String>,
}

#[derive(Template)]
#[template(path = "admin_settings.html")]
struct AdminSettingsTemplate {
    blog_title: String,
    blog_description: String,
    intro_content: String,
    nav_links: String,
    custom_css: String,
    default_css: String,
    favicon_url: String,
    og_image_url: String,
    custom_header: String,
    custom_footer: String,
    success: bool,
}

#[derive(Template)]
#[template(path = "posts.html")]
struct PostsListTemplate {
    blog_title: String,
    nav_links: Vec<NavLink>,
    posts: Vec<Post>,
    favicon_url: String,
    og_image_url: String,
    site_url: String,
    header_html: String,
    footer_html: String,
}

#[derive(Template)]
#[template(path = "admin_files.html")]
struct AdminFilesTemplate {
    files: Vec<UploadedFile>,
    site_url: String,
    error: Option<String>,
    success: bool,
}

// --- Query/Form structs ---

#[derive(serde::Deserialize, Default)]
pub struct FlashQuery {
    pub error: Option<String>,
    #[serde(default)]
    pub success: bool,
}

#[derive(serde::Deserialize)]
struct LoginForm {
    password: String,
}

#[derive(serde::Deserialize)]
struct PostForm {
    attributes: String,
    content: String,
    #[serde(default)]
    action: String,
}

struct ParsedAttributes {
    title: String,
    slug: String,
    alias: String,
    published_date: String,
    meta_description: String,
    meta_image: String,
    lang: String,
    tags: String,
}

fn parse_attributes(text: &str) -> ParsedAttributes {
    let mut attrs = ParsedAttributes {
        title: String::new(),
        slug: String::new(),
        alias: String::new(),
        published_date: String::new(),
        meta_description: String::new(),
        meta_image: String::new(),
        lang: String::new(),
        tags: String::new(),
    };
    for line in text.lines() {
        if let Some((key, value)) = line.split_once(':') {
            let key = key.trim().to_lowercase();
            let value = value.trim().to_string();
            match key.as_str() {
                "title" => attrs.title = value,
                "slug" => attrs.slug = value,
                "alias" => attrs.alias = value,
                "published_date" => attrs.published_date = value,
                "description" | "meta_description" => attrs.meta_description = value,
                "meta_image" => attrs.meta_image = value,
                "lang" => attrs.lang = value,
                "tags" => attrs.tags = value,
                _ => {}
            }
        }
    }
    attrs
}

#[derive(serde::Deserialize)]
struct PageForm {
    attributes: String,
    content: String,
}

struct ParsedPageAttributes {
    title: String,
    slug: String,
    is_published: bool,
}

fn parse_page_attributes(text: &str) -> ParsedPageAttributes {
    let mut attrs = ParsedPageAttributes {
        title: String::new(),
        slug: String::new(),
        is_published: false,
    };
    for line in text.lines() {
        if let Some((key, value)) = line.split_once(':') {
            let key = key.trim().to_lowercase();
            let value = value.trim().to_string();
            match key.as_str() {
                "title" => attrs.title = value,
                "slug" => attrs.slug = value,
                "published" => attrs.is_published = value == "true",
                _ => {}
            }
        }
    }
    attrs
}

#[derive(serde::Deserialize)]
struct SettingsForm {
    blog_title: String,
    blog_description: String,
    intro_content: String,
    nav_links: String,
    custom_css: String,
    favicon_url: String,
    og_image_url: String,
    custom_header: String,
    custom_footer: String,
}

// --- Helpers ---

fn mime_from_path(path: &str) -> &'static str {
    match path.rsplit('.').next().unwrap_or("") {
        "css" => "text/css",
        "js" => "application/javascript",
        "html" => "text/html",
        "png" => "image/png",
        "jpg" | "jpeg" => "image/jpeg",
        "gif" => "image/gif",
        "webp" => "image/webp",
        "ico" => "image/x-icon",
        "svg" => "image/svg+xml",
        "woff" | "woff2" => "font/woff2",
        "ttf" => "font/ttf",
        "otf" => "font/otf",
        "json" | "webmanifest" => "application/json",
        "pdf" => "application/pdf",
        "mp4" => "video/mp4",
        "webm" => "video/webm",
        _ => "application/octet-stream",
    }
}

fn get_header_footer_html(db: &db::Db) -> (String, String) {
    let custom_header = db::get_setting(db, "custom_header")
        .ok()
        .flatten()
        .unwrap_or_default();
    let custom_footer = db::get_setting(db, "custom_footer")
        .ok()
        .flatten()
        .unwrap_or_default();
    let header_html = render_markdown(&custom_header);
    let footer_html = render_markdown(&custom_footer);
    (header_html, footer_html)
}

fn render_markdown(content: &str) -> String {
    let mut options = Options::empty();
    options.insert(Options::ENABLE_STRIKETHROUGH);
    options.insert(Options::ENABLE_TABLES);
    options.insert(Options::ENABLE_TASKLISTS);
    options.insert(Options::ENABLE_FOOTNOTES);
    let parser = Parser::new_ext(content, options);
    let mut html_output = String::new();
    html::push_html(&mut html_output, parser);
    html_output
}

fn now_datetime() -> String {
    andromeda_auth::datetime::now_datetime_string()
}

fn slugify(s: &str) -> String {
    s.to_lowercase()
        .chars()
        .map(|c| if c.is_ascii_alphanumeric() { c } else { '-' })
        .collect::<String>()
        .split('-')
        .filter(|s| !s.is_empty())
        .collect::<Vec<_>>()
        .join("-")
}

fn opt_str(s: &str) -> Option<&str> {
    let trimmed = s.trim();
    if trimmed.is_empty() { None } else { Some(trimmed) }
}

fn get_blog_title(db: &Db) -> String {
    db::get_setting(db, "blog_title")
        .ok()
        .flatten()
        .unwrap_or_else(|| "My Blog".to_string())
}

fn parse_nav_links(input: &str) -> Vec<NavLink> {
    let mut links = Vec::new();
    let mut chars = input.chars().peekable();
    while let Some(c) = chars.next() {
        if c == '[' {
            let label: String = chars.by_ref().take_while(|&ch| ch != ']').collect();
            if chars.peek() == Some(&'(') {
                chars.next();
                let url: String = chars.by_ref().take_while(|&ch| ch != ')').collect();
                if !label.is_empty() && !url.is_empty() {
                    links.push(NavLink { label, url });
                }
            }
        }
    }
    links
}

fn get_nav_links(db: &Db) -> Vec<NavLink> {
    let raw = db::get_setting(db, "nav_links")
        .ok()
        .flatten()
        .unwrap_or_default();
    parse_nav_links(&raw)
}

fn get_favicon_url(db: &Db) -> String {
    db::get_setting(db, "favicon_url")
        .ok()
        .flatten()
        .unwrap_or_default()
}

fn get_og_image_url(db: &Db) -> String {
    db::get_setting(db, "og_image_url")
        .ok()
        .flatten()
        .unwrap_or_default()
}

fn render_latest_posts_embed(posts: &[&Post]) -> String {
    let mut html = String::from("<div class=\"post-list\">");
    for post in posts {
        html.push_str(&format!(
            r#"<a href="/posts/{slug}" class="post-item"><div class="post-item-info"><span class="post-title">{title}</span>"#,
            slug = post.slug,
            title = post.title,
        ));
        if let Some(ref tags) = post.tags {
            if !tags.is_empty() {
                html.push_str(r#"<span class="post-tags">"#);
                for tag in tags.split(',') {
                    let tag = tag.trim();
                    if !tag.is_empty() {
                        html.push_str(&format!(r#"<span class="tag">{}</span>"#, tag));
                    }
                }
                html.push_str("</span>");
            }
        }
        html.push_str("</div>");
        if let Some(ref date) = post.published_date {
            html.push_str(&format!(r#"<time class="post-date">{}</time>"#, date));
        }
        html.push_str("</a>");
    }
    html.push_str("</div>");
    html
}

// --- Router ---

pub async fn run(host: String, port: u16) {
    use handlers::{admin, public};

    dotenvy::dotenv().ok();

    let db = db::init_db();

    if let Err(e) = db::prune_expired_sessions(&db) {
        tracing::warn!("Failed to prune sessions: {}", e);
    }

    let app_password = std::env::var("POSTS_PASSWORD").unwrap_or_else(|_| {
        tracing::warn!("POSTS_PASSWORD not set, using default 'changeme'");
        "changeme".to_string()
    });

    let cookie_secure = std::env::var("COOKIE_SECURE")
        .map(|v| v == "true")
        .unwrap_or(false);

    let uploads_dir = std::env::var("UPLOADS_DIR").unwrap_or_else(|_| "uploads".to_string());
    tokio::fs::create_dir_all(&uploads_dir)
        .await
        .expect("Failed to create uploads directory");

    let site_url = std::env::var("SITE_URL")
        .unwrap_or_else(|_| "http://localhost:3000".to_string())
        .trim_end_matches('/')
        .to_string();

    let state = Arc::new(AppState {
        db,
        app_password,
        cookie_secure,
        uploads_dir,
        site_url,
    });

    let app = Router::new()
        // Public routes
        .route("/", get(public::public_index))
        .route("/posts", get(public::public_posts_list))
        .route("/posts/{slug}", get(public::public_post))
        .route("/custom-styles.css", get(public::serve_custom_css))
        .route("/{slug}", get(public::public_page))
        .route("/feed.xml", get(public::rss_feed))
        // Admin auth
        .route("/admin/login", get(admin::get_login).post(admin::post_login))
        .route("/admin/logout", get(admin::get_logout))
        // Admin posts
        .route("/admin", get(admin::admin_index))
        .route("/admin/posts/new", get(admin::admin_new_post))
        .route("/admin/posts", post(admin::admin_create_post))
        .route("/admin/posts/{id}/edit", get(admin::admin_edit_post))
        .route("/admin/posts/{id}", post(admin::admin_update_post))
        .route("/admin/posts/{id}/delete", post(admin::admin_delete_post))
        .route("/admin/posts/{id}/publish", post(admin::admin_toggle_publish))
        // Admin pages
        .route("/admin/pages", get(admin::admin_pages))
        .route("/admin/pages/new", get(admin::admin_new_page))
        .route("/admin/pages/create", post(admin::admin_create_page))
        .route("/admin/pages/{id}/edit", get(admin::admin_edit_page))
        .route("/admin/pages/{id}", post(admin::admin_update_page))
        .route("/admin/pages/{id}/delete", post(admin::admin_delete_page))
        // Admin settings
        .route(
            "/admin/settings",
            get(admin::admin_get_settings).post(admin::admin_post_settings),
        )
        // Admin downloads
        .route("/admin/downloads/posts", get(admin::admin_download_posts))
        .route("/admin/downloads/uploads", get(admin::admin_download_uploads))
        // Admin files
        .route("/admin/files", get(admin::admin_files))
        .route("/admin/files/upload", post(admin::admin_upload_file))
        .route("/admin/files/{id}/delete", post(admin::admin_delete_file))
        // Public files
        .route("/files/{filename}", get(public::serve_uploaded_file))
        // Static assets
        .route("/static/{*path}", get(public::serve_static))
        // Fallback
        .fallback(get(public::fallback_handler))
        .with_state(state)
        .layer(DefaultBodyLimit::max(11 * 1024 * 1024));

    let addr = format!("{}:{}", host, port);
    tracing::info!("Listening on http://{}", addr);

    let listener = tokio::net::TcpListener::bind(&addr).await.unwrap();
    axum::serve(listener, app).await.unwrap();
}
