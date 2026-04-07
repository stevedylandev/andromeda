use askama::Template;
use askama_web::WebTemplate;
use axum::{
    extract::{Form, Path, Query, State},
    http::{HeaderValue, StatusCode, Uri},
    response::{Html, IntoResponse, Redirect, Response},
    routing::{get, post},
    Router,
};
use pulldown_cmark::{Options, Parser, html};
use rust_embed::Embed;
use std::sync::Arc;

use crate::auth;
use crate::db::{self, Db, Page, Post};

#[derive(Clone)]
pub struct AppState {
    pub db: Db,
    pub app_password: String,
    pub cookie_secure: bool,
}

#[derive(Embed)]
#[folder = "static/"]
struct Static;

// --- Templates ---

#[derive(Template)]
#[template(path = "base.html")]
struct BaseTemplate {
    blog_title: String,
    nav_pages: Vec<Page>,
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
    nav_pages: Vec<Page>,
}

#[derive(Template)]
#[template(path = "post.html")]
struct PostTemplate {
    blog_title: String,
    nav_pages: Vec<Page>,
    post: Post,
    rendered_content: String,
}

#[derive(Template)]
#[template(path = "page.html")]
struct PageTemplate {
    blog_title: String,
    nav_pages: Vec<Page>,
    page: Page,
    rendered_content: String,
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
                "meta_description" => attrs.meta_description = value,
                "meta_image" => attrs.meta_image = value,
                "lang" => attrs.lang = value,
                "tags" => attrs.tags = value,
                _ => {} // ignore unknown keys (including canonical_url)
            }
        }
    }
    attrs
}

#[derive(serde::Deserialize)]
struct PageForm {
    title: String,
    slug: String,
    content: String,
    #[serde(default)]
    is_published: Option<String>,
    #[serde(default)]
    nav_order: i64,
}

#[derive(serde::Deserialize)]
struct SettingsForm {
    blog_title: String,
    blog_description: String,
    intro_content: String,
}

// --- Helpers ---

fn mime_from_path(path: &str) -> &'static str {
    match path.rsplit('.').next().unwrap_or("") {
        "css" => "text/css",
        "js" => "application/javascript",
        "html" => "text/html",
        "png" => "image/png",
        "ico" => "image/x-icon",
        "svg" => "image/svg+xml",
        "woff" | "woff2" => "font/woff2",
        "ttf" => "font/ttf",
        "otf" => "font/otf",
        "json" | "webmanifest" => "application/json",
        _ => "application/octet-stream",
    }
}

fn render_markdown(content: &str) -> String {
    let mut options = Options::empty();
    options.insert(Options::ENABLE_STRIKETHROUGH);
    options.insert(Options::ENABLE_TABLES);
    options.insert(Options::ENABLE_TASKLISTS);
    let parser = Parser::new_ext(content, options);
    let mut html_output = String::new();
    html::push_html(&mut html_output, parser);
    html_output
}

fn now_datetime() -> String {
    use std::time::{SystemTime, UNIX_EPOCH};
    let secs = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .unwrap()
        .as_secs();
    let days = secs / 86400;
    let tod = secs % 86400;
    let (y, m, d) = days_to_ymd(days as i64);
    format!(
        "{:04}-{:02}-{:02} {:02}:{:02}:{:02}",
        y, m, d, tod / 3600, (tod % 3600) / 60, tod % 60
    )
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

fn get_nav_pages(db: &Db) -> Vec<Page> {
    db::get_published_pages(db).unwrap_or_default()
}

// --- Static file handler ---

async fn serve_static(Path(path): Path<String>) -> Response {
    match Static::get(&path) {
        Some(file) => {
            let mime = mime_from_path(&path);
            (
                StatusCode::OK,
                [(axum::http::header::CONTENT_TYPE, HeaderValue::from_static(mime))],
                file.data.to_vec(),
            )
                .into_response()
        }
        None => StatusCode::NOT_FOUND.into_response(),
    }
}

// --- Auth handlers ---

async fn get_login(Query(q): Query<FlashQuery>) -> Response {
    WebTemplate(LoginTemplate { error: q.error }).into_response()
}

async fn post_login(
    State(state): State<Arc<AppState>>,
    Form(form): Form<LoginForm>,
) -> Response {
    if !auth::verify_password(&form.password, &state.app_password) {
        return Redirect::to("/admin/login?error=Invalid+password").into_response();
    }

    let token = auth::generate_session_token();

    let expires_at = {
        use std::time::{SystemTime, UNIX_EPOCH};
        let secs = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap()
            .as_secs()
            + 7 * 24 * 3600;
        let days = secs / 86400;
        let tod = secs % 86400;
        let (y, m, d) = days_to_ymd(days as i64);
        format!(
            "{:04}-{:02}-{:02} {:02}:{:02}:{:02}",
            y, m, d, tod / 3600, (tod % 3600) / 60, tod % 60
        )
    };

    if let Err(e) = db::insert_session(&state.db, &token, &expires_at) {
        tracing::error!("Failed to create session: {}", e);
        return Redirect::to("/admin/login?error=Server+error").into_response();
    }

    let cookie = auth::build_session_cookie(&token, state.cookie_secure);
    let mut resp = Redirect::to("/admin").into_response();
    resp.headers_mut().insert(
        axum::http::header::SET_COOKIE,
        HeaderValue::from_str(&cookie).unwrap(),
    );
    resp
}

async fn get_logout(State(state): State<Arc<AppState>>, headers: axum::http::HeaderMap) -> Response {
    if let Some(cookie_header) = headers.get("cookie").and_then(|v| v.to_str().ok()) {
        for part in cookie_header.split(';') {
            let part = part.trim();
            if let Some(val) = part.strip_prefix("session=") {
                let val = val.trim();
                if !val.is_empty() {
                    let _ = db::delete_session(&state.db, val);
                }
            }
        }
    }

    let cookie = auth::clear_session_cookie();
    let mut resp = Redirect::to("/admin/login").into_response();
    resp.headers_mut().insert(
        axum::http::header::SET_COOKIE,
        HeaderValue::from_str(&cookie).unwrap(),
    );
    resp
}

// --- Public handlers ---

async fn public_index(State(state): State<Arc<AppState>>) -> Response {
    let blog_title = get_blog_title(&state.db);
    let blog_description = db::get_setting(&state.db, "blog_description")
        .ok()
        .flatten()
        .unwrap_or_default();
    let intro_content = db::get_setting(&state.db, "intro_content")
        .ok()
        .flatten()
        .unwrap_or_default();
    let intro_html = render_markdown(&intro_content);
    let nav_pages = get_nav_pages(&state.db);

    match db::get_published_posts(&state.db) {
        Ok(posts) => WebTemplate(IndexTemplate {
            blog_title,
            blog_description,
            intro_html,
            posts,
            nav_pages,
        })
        .into_response(),
        Err(e) => {
            tracing::error!("Failed to list posts: {}", e);
            (StatusCode::INTERNAL_SERVER_ERROR, Html("Server error".to_string())).into_response()
        }
    }
}

async fn public_post(
    State(state): State<Arc<AppState>>,
    Path(slug): Path<String>,
) -> Response {
    match db::get_post_by_slug(&state.db, &slug) {
        Ok(Some(post)) if post.status == "published" => {
            let rendered_content = render_markdown(&post.content);
            let blog_title = get_blog_title(&state.db);
            let nav_pages = get_nav_pages(&state.db);
            WebTemplate(PostTemplate {
                blog_title,
                nav_pages,
                post,
                rendered_content,
            })
            .into_response()
        }
        Ok(_) => (StatusCode::NOT_FOUND, Html("Not found".to_string())).into_response(),
        Err(e) => {
            tracing::error!("Failed to get post: {}", e);
            (StatusCode::INTERNAL_SERVER_ERROR, Html("Server error".to_string())).into_response()
        }
    }
}

async fn public_page(
    State(state): State<Arc<AppState>>,
    Path(slug): Path<String>,
) -> Response {
    match db::get_page_by_slug(&state.db, &slug) {
        Ok(Some(page)) if page.is_published => {
            let rendered_content = render_markdown(&page.content);
            let blog_title = get_blog_title(&state.db);
            let nav_pages = get_nav_pages(&state.db);
            WebTemplate(PageTemplate {
                blog_title,
                nav_pages,
                page,
                rendered_content,
            })
            .into_response()
        }
        Ok(_) => (StatusCode::NOT_FOUND, Html("Not found".to_string())).into_response(),
        Err(e) => {
            tracing::error!("Failed to get page: {}", e);
            (StatusCode::INTERNAL_SERVER_ERROR, Html("Server error".to_string())).into_response()
        }
    }
}

async fn fallback_handler(
    State(state): State<Arc<AppState>>,
    uri: Uri,
) -> Response {
    let path = uri.path().trim_start_matches('/');
    if let Ok(Some(redirect_to)) = db::find_alias_redirect(&state.db, path) {
        return Redirect::permanent(&redirect_to).into_response();
    }
    (StatusCode::NOT_FOUND, Html("Not found".to_string())).into_response()
}

// --- Admin post handlers ---

async fn admin_index(
    _session: auth::AuthSession,
    State(state): State<Arc<AppState>>,
) -> Response {
    match db::get_all_posts(&state.db) {
        Ok(posts) => WebTemplate(AdminIndexTemplate { posts }).into_response(),
        Err(e) => {
            tracing::error!("Failed to list posts: {}", e);
            (StatusCode::INTERNAL_SERVER_ERROR, Html("Server error".to_string())).into_response()
        }
    }
}

async fn admin_new_post(
    _session: auth::AuthSession,
    Query(q): Query<FlashQuery>,
) -> Response {
    WebTemplate(AdminPostFormTemplate {
        post: None,
        error: q.error,
    })
    .into_response()
}

async fn admin_create_post(
    _session: auth::AuthSession,
    State(state): State<Arc<AppState>>,
    Form(form): Form<PostForm>,
) -> Response {
    let attrs = parse_attributes(&form.attributes);
    let title = attrs.title.trim();
    if title.is_empty() {
        return Redirect::to("/admin/posts/new?error=Title+is+required").into_response();
    }
    let slug = if attrs.slug.trim().is_empty() {
        slugify(title)
    } else {
        attrs.slug.trim().to_string()
    };

    let status = if form.action == "publish" { "published" } else { "draft" };
    let lang = if attrs.lang.trim().is_empty() { "en" } else { attrs.lang.trim() };
    let published_date = if attrs.published_date.trim().is_empty() {
        now_datetime()
    } else {
        attrs.published_date.trim().to_string()
    };

    match db::create_post(
        &state.db,
        title,
        &slug,
        &form.content,
        status,
        opt_str(&attrs.alias),
        None,
        Some(&published_date),
        opt_str(&attrs.meta_description),
        opt_str(&attrs.meta_image),
        lang,
        opt_str(&attrs.tags),
    ) {
        Ok(_) => Redirect::to("/admin").into_response(),
        Err(e) => {
            tracing::error!("Failed to create post: {}", e);
            Redirect::to("/admin/posts/new?error=Failed+to+create+post").into_response()
        }
    }
}

async fn admin_edit_post(
    _session: auth::AuthSession,
    State(state): State<Arc<AppState>>,
    Path(short_id): Path<String>,
    Query(q): Query<FlashQuery>,
) -> Response {
    match db::get_post_by_short_id(&state.db, &short_id) {
        Ok(Some(post)) => WebTemplate(AdminPostFormTemplate {
            post: Some(post),
            error: q.error,
        })
        .into_response(),
        Ok(None) => (StatusCode::NOT_FOUND, Html("Post not found".to_string())).into_response(),
        Err(e) => {
            tracing::error!("Failed to get post: {}", e);
            (StatusCode::INTERNAL_SERVER_ERROR, Html("Server error".to_string())).into_response()
        }
    }
}

async fn admin_update_post(
    _session: auth::AuthSession,
    State(state): State<Arc<AppState>>,
    Path(short_id): Path<String>,
    Form(form): Form<PostForm>,
) -> Response {
    let attrs = parse_attributes(&form.attributes);
    let title = attrs.title.trim();
    if title.is_empty() {
        return Redirect::to(&format!("/admin/posts/{}/edit?error=Title+is+required", short_id))
            .into_response();
    }
    let slug = if attrs.slug.trim().is_empty() {
        slugify(title)
    } else {
        attrs.slug.trim().to_string()
    };

    let lang = if attrs.lang.trim().is_empty() { "en" } else { attrs.lang.trim() };
    let published_date = if attrs.published_date.trim().is_empty() {
        now_datetime()
    } else {
        attrs.published_date.trim().to_string()
    };

    match db::update_post(
        &state.db,
        &short_id,
        title,
        &slug,
        &form.content,
        opt_str(&attrs.alias),
        None,
        Some(&published_date),
        opt_str(&attrs.meta_description),
        opt_str(&attrs.meta_image),
        lang,
        opt_str(&attrs.tags),
    ) {
        Ok(Some(_)) => Redirect::to("/admin").into_response(),
        Ok(None) => (StatusCode::NOT_FOUND, Html("Post not found".to_string())).into_response(),
        Err(e) => {
            tracing::error!("Failed to update post: {}", e);
            Redirect::to(&format!("/admin/posts/{}/edit?error=Failed+to+update", short_id))
                .into_response()
        }
    }
}

async fn admin_delete_post(
    _session: auth::AuthSession,
    State(state): State<Arc<AppState>>,
    Path(short_id): Path<String>,
) -> Response {
    match db::delete_post(&state.db, &short_id) {
        Ok(_) => Redirect::to("/admin").into_response(),
        Err(e) => {
            tracing::error!("Failed to delete post: {}", e);
            Redirect::to("/admin").into_response()
        }
    }
}

async fn admin_toggle_publish(
    _session: auth::AuthSession,
    State(state): State<Arc<AppState>>,
    Path(short_id): Path<String>,
) -> Response {
    match db::toggle_post_status(&state.db, &short_id) {
        Ok(_) => Redirect::to("/admin").into_response(),
        Err(e) => {
            tracing::error!("Failed to toggle post status: {}", e);
            Redirect::to("/admin").into_response()
        }
    }
}

// --- Admin page handlers ---

async fn admin_pages(
    _session: auth::AuthSession,
    State(state): State<Arc<AppState>>,
) -> Response {
    match db::get_all_pages(&state.db) {
        Ok(pages) => WebTemplate(AdminPagesTemplate { pages }).into_response(),
        Err(e) => {
            tracing::error!("Failed to list pages: {}", e);
            (StatusCode::INTERNAL_SERVER_ERROR, Html("Server error".to_string())).into_response()
        }
    }
}

async fn admin_new_page(
    _session: auth::AuthSession,
    Query(q): Query<FlashQuery>,
) -> Response {
    WebTemplate(AdminPageFormTemplate {
        page: None,
        error: q.error,
    })
    .into_response()
}

async fn admin_create_page(
    _session: auth::AuthSession,
    State(state): State<Arc<AppState>>,
    Form(form): Form<PageForm>,
) -> Response {
    let title = form.title.trim();
    let slug = form.slug.trim();
    if title.is_empty() || slug.is_empty() {
        return Redirect::to("/admin/pages/new?error=Title+and+slug+are+required").into_response();
    }

    let is_published = form.is_published.as_deref() == Some("on");

    match db::create_page(&state.db, title, slug, &form.content, is_published, form.nav_order) {
        Ok(_) => Redirect::to("/admin/pages").into_response(),
        Err(e) => {
            tracing::error!("Failed to create page: {}", e);
            Redirect::to("/admin/pages/new?error=Failed+to+create+page").into_response()
        }
    }
}

async fn admin_edit_page(
    _session: auth::AuthSession,
    State(state): State<Arc<AppState>>,
    Path(short_id): Path<String>,
    Query(q): Query<FlashQuery>,
) -> Response {
    match db::get_page_by_short_id(&state.db, &short_id) {
        Ok(Some(page)) => WebTemplate(AdminPageFormTemplate {
            page: Some(page),
            error: q.error,
        })
        .into_response(),
        Ok(None) => (StatusCode::NOT_FOUND, Html("Page not found".to_string())).into_response(),
        Err(e) => {
            tracing::error!("Failed to get page: {}", e);
            (StatusCode::INTERNAL_SERVER_ERROR, Html("Server error".to_string())).into_response()
        }
    }
}

async fn admin_update_page(
    _session: auth::AuthSession,
    State(state): State<Arc<AppState>>,
    Path(short_id): Path<String>,
    Form(form): Form<PageForm>,
) -> Response {
    let title = form.title.trim();
    let slug = form.slug.trim();
    if title.is_empty() || slug.is_empty() {
        return Redirect::to(&format!("/admin/pages/{}/edit?error=Title+and+slug+are+required", short_id))
            .into_response();
    }

    let is_published = form.is_published.as_deref() == Some("on");

    match db::update_page(&state.db, &short_id, title, slug, &form.content, is_published, form.nav_order) {
        Ok(Some(_)) => Redirect::to("/admin/pages").into_response(),
        Ok(None) => (StatusCode::NOT_FOUND, Html("Page not found".to_string())).into_response(),
        Err(e) => {
            tracing::error!("Failed to update page: {}", e);
            Redirect::to(&format!("/admin/pages/{}/edit?error=Failed+to+update", short_id))
                .into_response()
        }
    }
}

async fn admin_delete_page(
    _session: auth::AuthSession,
    State(state): State<Arc<AppState>>,
    Path(short_id): Path<String>,
) -> Response {
    match db::delete_page(&state.db, &short_id) {
        Ok(_) => Redirect::to("/admin/pages").into_response(),
        Err(e) => {
            tracing::error!("Failed to delete page: {}", e);
            Redirect::to("/admin/pages").into_response()
        }
    }
}

// --- Admin settings handlers ---

async fn admin_get_settings(
    _session: auth::AuthSession,
    State(state): State<Arc<AppState>>,
    Query(q): Query<FlashQuery>,
) -> Response {
    let blog_title = db::get_setting(&state.db, "blog_title").ok().flatten().unwrap_or_default();
    let blog_description = db::get_setting(&state.db, "blog_description").ok().flatten().unwrap_or_default();
    let intro_content = db::get_setting(&state.db, "intro_content").ok().flatten().unwrap_or_default();

    WebTemplate(AdminSettingsTemplate {
        blog_title,
        blog_description,
        intro_content,
        success: q.success,
    })
    .into_response()
}

async fn admin_post_settings(
    _session: auth::AuthSession,
    State(state): State<Arc<AppState>>,
    Form(form): Form<SettingsForm>,
) -> Response {
    let _ = db::set_setting(&state.db, "blog_title", form.blog_title.trim());
    let _ = db::set_setting(&state.db, "blog_description", form.blog_description.trim());
    let _ = db::set_setting(&state.db, "intro_content", &form.intro_content);
    Redirect::to("/admin/settings?success=true").into_response()
}

// --- Date helper ---

fn days_to_ymd(mut days: i64) -> (i64, i64, i64) {
    days += 719468;
    let era = if days >= 0 { days } else { days - 146096 } / 146097;
    let doe = (days - era * 146097) as u32;
    let yoe = (doe - doe / 1460 + doe / 36524 - doe / 146096) / 365;
    let y = yoe as i64 + era * 400;
    let doy = doe - (365 * yoe + yoe / 4 - yoe / 100);
    let mp = (5 * doy + 2) / 153;
    let d = doy - (153 * mp + 2) / 5 + 1;
    let m = if mp < 10 { mp + 3 } else { mp - 9 };
    let y = if m <= 2 { y + 1 } else { y };
    (y, m as i64, d as i64)
}

// --- Router ---

pub async fn run(host: String, port: u16) {
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

    let state = Arc::new(AppState {
        db,
        app_password,
        cookie_secure,
    });

    let app = Router::new()
        // Public routes
        .route("/", get(public_index))
        .route("/posts/{slug}", get(public_post))
        .route("/pages/{slug}", get(public_page))
        // Admin auth
        .route("/admin/login", get(get_login).post(post_login))
        .route("/admin/logout", get(get_logout))
        // Admin posts
        .route("/admin", get(admin_index))
        .route("/admin/posts/new", get(admin_new_post))
        .route("/admin/posts", post(admin_create_post))
        .route("/admin/posts/{id}/edit", get(admin_edit_post))
        .route("/admin/posts/{id}", post(admin_update_post))
        .route("/admin/posts/{id}/delete", post(admin_delete_post))
        .route("/admin/posts/{id}/publish", post(admin_toggle_publish))
        // Admin pages
        .route("/admin/pages", get(admin_pages))
        .route("/admin/pages/new", get(admin_new_page))
        .route("/admin/pages/create", post(admin_create_page))
        .route("/admin/pages/{id}/edit", get(admin_edit_page))
        .route("/admin/pages/{id}", post(admin_update_page))
        .route("/admin/pages/{id}/delete", post(admin_delete_page))
        // Admin settings
        .route("/admin/settings", get(admin_get_settings).post(admin_post_settings))
        // Static assets
        .route("/static/{*path}", get(serve_static))
        // Fallback
        .fallback(get(fallback_handler))
        .with_state(state);

    let addr = format!("{}:{}", host, port);
    tracing::info!("Listening on http://{}", addr);

    let listener = tokio::net::TcpListener::bind(&addr).await.unwrap();
    axum::serve(listener, app).await.unwrap();
}
