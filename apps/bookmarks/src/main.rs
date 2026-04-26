mod auth;
mod db;

use std::sync::{Arc, Mutex};

use andromeda_db::{
    Db,
    session::{SESSION_SCHEMA, prune_expired_sessions},
};
use askama::Template;
use axum::{
    Form, Json, Router,
    extract::{Path, Query, Request, State},
    http::{HeaderMap, StatusCode, header},
    middleware::{self, Next},
    response::{Html, IntoResponse, Redirect, Response},
    routing::{get, post},
};
use rusqlite::Connection;
use rust_embed::Embed;
use serde::Deserialize;

#[derive(Embed)]
#[folder = "static/"]
struct Static;

async fn static_handler(Path(path): Path<String>) -> Response {
    match Static::get(&path) {
        Some(file) => {
            let mime = mime_guess::from_path(&path).first_or_octet_stream();
            ([(header::CONTENT_TYPE, mime.as_ref())], file.data.to_vec()).into_response()
        }
        None => StatusCode::NOT_FOUND.into_response(),
    }
}

use crate::db::{Category, Link};

pub struct AppState {
    pub db: Db,
    pub admin_password: Option<String>,
    pub api_key: Option<String>,
    pub cookie_secure: bool,
}

// ── Templates ────────────────────────────────────────────────────────────

struct CategoryGroup {
    name: String,
    links: Vec<Link>,
}

#[derive(Template)]
#[template(path = "index.html")]
struct IndexTemplate {
    groups: Vec<CategoryGroup>,
}

#[derive(Template)]
#[template(path = "login.html")]
struct LoginTemplate {
    error: Option<String>,
}

#[derive(Template)]
#[template(path = "admin.html")]
struct AdminTemplate {
    success: Option<String>,
    error: Option<String>,
    categories: Vec<Category>,
    links: Vec<AdminLinkRow>,
}

struct AdminLinkRow {
    short_id: String,
    title: String,
    url: String,
    category: String,
}

#[derive(Deserialize, Default)]
struct FlashQuery {
    error: Option<String>,
    success: Option<String>,
}

fn render<T: Template>(tpl: T) -> Response {
    match tpl.render() {
        Ok(html) => Html(html).into_response(),
        Err(e) => {
            tracing::error!("template render: {e}");
            (StatusCode::INTERNAL_SERVER_ERROR, "Template error").into_response()
        }
    }
}

// ── Public web ───────────────────────────────────────────────────────────

async fn index_handler(State(state): State<Arc<AppState>>) -> Response {
    let categories = db::list_categories(&state.db).unwrap_or_default();
    let all_links = db::list_links(&state.db).unwrap_or_default();
    let groups = categories
        .into_iter()
        .map(|c| {
            let links = all_links
                .iter()
                .filter(|l| l.category_id == c.id)
                .cloned()
                .collect();
            CategoryGroup { name: c.name, links }
        })
        .collect();
    render(IndexTemplate { groups })
}

// ── Login / logout ───────────────────────────────────────────────────────

#[derive(Deserialize)]
struct LoginForm {
    password: String,
}

async fn login_get(Query(q): Query<FlashQuery>) -> Response {
    render(LoginTemplate { error: q.error })
}

async fn login_post(State(state): State<Arc<AppState>>, Form(form): Form<LoginForm>) -> Response {
    let pw = match &state.admin_password {
        Some(p) => p,
        None => return Redirect::to("/login?error=No+password+configured").into_response(),
    };
    if !auth::verify_password(&form.password, pw) {
        return Redirect::to("/login?error=Invalid+password").into_response();
    }
    let token = auth::generate_session_token();
    if let Err(e) = auth::create_session(&state.db, &token) {
        tracing::error!("create session: {e}");
        return Redirect::to("/login?error=Session+error").into_response();
    }
    let _ = prune_expired_sessions(&state.db);
    let cookie = auth::build_session_cookie(&token, state.cookie_secure);
    let mut resp = Redirect::to("/admin").into_response();
    resp.headers_mut()
        .insert(header::SET_COOKIE, cookie.parse().unwrap());
    resp
}

async fn logout_handler(State(state): State<Arc<AppState>>, headers: HeaderMap) -> Response {
    if let Some(token) = auth::extract_session_cookie(&headers) {
        auth::delete_session(&state.db, &token);
    }
    let mut resp = Redirect::to("/login").into_response();
    resp.headers_mut()
        .insert(header::SET_COOKIE, auth::clear_session_cookie().parse().unwrap());
    resp
}

// ── Admin ────────────────────────────────────────────────────────────────

async fn admin_handler(
    _session: auth::AuthSession,
    State(state): State<Arc<AppState>>,
    Query(q): Query<FlashQuery>,
) -> Response {
    let categories = db::list_categories(&state.db).unwrap_or_default();
    let raw_links = db::list_links(&state.db).unwrap_or_default();
    let links = raw_links
        .into_iter()
        .map(|l| {
            let cat = categories
                .iter()
                .find(|c| c.id == l.category_id)
                .map(|c| c.name.clone())
                .unwrap_or_default();
            AdminLinkRow {
                short_id: l.short_id,
                title: l.title,
                url: l.url,
                category: cat,
            }
        })
        .collect();
    render(AdminTemplate {
        success: q.success,
        error: q.error,
        categories,
        links,
    })
}

#[derive(Deserialize)]
struct AddCategoryForm {
    name: String,
}

async fn admin_add_category(
    _session: auth::AuthSession,
    State(state): State<Arc<AppState>>,
    Form(form): Form<AddCategoryForm>,
) -> Response {
    let name = form.name.trim();
    if name.is_empty() {
        return Redirect::to("/admin?error=Name+required").into_response();
    }
    match db::create_category(&state.db, name) {
        Ok(_) => Redirect::to("/admin?success=Category+added").into_response(),
        Err(e) => {
            tracing::error!("create category: {e}");
            Redirect::to("/admin?error=Failed+to+add+category").into_response()
        }
    }
}

async fn admin_delete_category(
    _session: auth::AuthSession,
    State(state): State<Arc<AppState>>,
    Path(short_id): Path<String>,
) -> Response {
    let _ = db::delete_category_by_short_id(&state.db, &short_id);
    Redirect::to("/admin?success=Category+removed").into_response()
}

#[derive(Deserialize)]
struct AddLinkForm {
    title: String,
    url: String,
    category: String,
}

async fn admin_add_link(
    _session: auth::AuthSession,
    State(state): State<Arc<AppState>>,
    Form(form): Form<AddLinkForm>,
) -> Response {
    let title = form.title.trim();
    let url = form.url.trim();
    if title.is_empty() || url.is_empty() {
        return Redirect::to("/admin?error=Title+and+URL+required").into_response();
    }
    let cat = match db::get_category_by_name(&state.db, form.category.trim()) {
        Ok(Some(c)) => c,
        Ok(None) => return Redirect::to("/admin?error=Unknown+category").into_response(),
        Err(e) => {
            tracing::error!("get category: {e}");
            return Redirect::to("/admin?error=Server+error").into_response();
        }
    };
    match db::create_link(&state.db, title, url, cat.id) {
        Ok(_) => Redirect::to("/admin?success=Link+added").into_response(),
        Err(e) => {
            tracing::error!("create link: {e}");
            Redirect::to("/admin?error=Failed+to+add+link").into_response()
        }
    }
}

async fn admin_delete_link(
    _session: auth::AuthSession,
    State(state): State<Arc<AppState>>,
    Path(short_id): Path<String>,
) -> Response {
    let _ = db::delete_link_by_short_id(&state.db, &short_id);
    Redirect::to("/admin?success=Link+removed").into_response()
}

// ── JSON API ─────────────────────────────────────────────────────────────

async fn api_list_categories(State(state): State<Arc<AppState>>) -> Response {
    match db::list_categories(&state.db) {
        Ok(cats) => Json(cats).into_response(),
        Err(e) => {
            tracing::error!("list categories: {e}");
            StatusCode::INTERNAL_SERVER_ERROR.into_response()
        }
    }
}

#[derive(Deserialize)]
struct ListLinksQuery {
    category: Option<String>,
}

async fn api_list_links(
    State(state): State<Arc<AppState>>,
    Query(q): Query<ListLinksQuery>,
) -> Response {
    let categories = match db::list_categories(&state.db) {
        Ok(c) => c,
        Err(e) => {
            tracing::error!("list categories: {e}");
            return StatusCode::INTERNAL_SERVER_ERROR.into_response();
        }
    };
    let links = match db::list_links(&state.db) {
        Ok(l) => l,
        Err(e) => {
            tracing::error!("list links: {e}");
            return StatusCode::INTERNAL_SERVER_ERROR.into_response();
        }
    };
    if let Some(name) = q.category.as_deref().map(str::trim).filter(|s| !s.is_empty()) {
        let Some(cat) = categories.iter().find(|c| c.name.eq_ignore_ascii_case(name)) else {
            return (
                StatusCode::NOT_FOUND,
                Json(serde_json::json!({ "error": "unknown category" })),
            )
                .into_response();
        };
        let filtered: Vec<&Link> = links.iter().filter(|l| l.category_id == cat.id).collect();
        return Json(filtered).into_response();
    }
    let mut grouped = serde_json::Map::new();
    for cat in &categories {
        let items: Vec<&Link> = links.iter().filter(|l| l.category_id == cat.id).collect();
        grouped.insert(cat.name.clone(), serde_json::to_value(items).unwrap());
    }
    Json(serde_json::Value::Object(grouped)).into_response()
}

#[derive(Deserialize)]
struct ApiCreateLink {
    category: String,
    title: String,
    url: String,
}

async fn api_create_link(
    State(state): State<Arc<AppState>>,
    Json(body): Json<ApiCreateLink>,
) -> Response {
    let title = body.title.trim();
    let url = body.url.trim();
    if title.is_empty() || url.is_empty() {
        return (
            StatusCode::BAD_REQUEST,
            Json(serde_json::json!({ "error": "title and url required" })),
        )
            .into_response();
    }
    let cat = match db::get_category_by_name(&state.db, body.category.trim()) {
        Ok(Some(c)) => c,
        Ok(None) => {
            return (
                StatusCode::NOT_FOUND,
                Json(serde_json::json!({ "error": "unknown category" })),
            )
                .into_response();
        }
        Err(e) => {
            tracing::error!("get category: {e}");
            return StatusCode::INTERNAL_SERVER_ERROR.into_response();
        }
    };
    match db::create_link(&state.db, title, url, cat.id) {
        Ok(link) => (StatusCode::CREATED, Json(link)).into_response(),
        Err(e) => {
            tracing::error!("create link: {e}");
            StatusCode::INTERNAL_SERVER_ERROR.into_response()
        }
    }
}

async fn require_api_key(
    State(state): State<Arc<AppState>>,
    headers: HeaderMap,
    request: Request,
    next: Next,
) -> Result<Response, (StatusCode, Json<serde_json::Value>)> {
    let server_key = state.api_key.as_deref().ok_or((
        StatusCode::FORBIDDEN,
        Json(serde_json::json!({ "error": "API key not configured" })),
    ))?;
    let provided = headers.get("x-api-key").and_then(|v| v.to_str().ok());
    if let Some(k) = provided {
        if auth::verify_api_key(k, server_key) {
            return Ok(next.run(request).await);
        }
    }
    Err((
        StatusCode::UNAUTHORIZED,
        Json(serde_json::json!({ "error": "Invalid or missing API key" })),
    ))
}

// ── main ─────────────────────────────────────────────────────────────────

#[tokio::main]
async fn main() {
    dotenvy::dotenv().ok();
    tracing_subscriber::fmt()
        .with_env_filter(
            tracing_subscriber::EnvFilter::try_from_default_env()
                .unwrap_or_else(|_| tracing_subscriber::EnvFilter::new("info,bookmarks=info")),
        )
        .init();

    let db_path =
        std::env::var("BOOKMARKS_DB_PATH").unwrap_or_else(|_| "bookmarks.sqlite".to_string());
    let conn = Connection::open(&db_path).expect("open sqlite");
    conn.execute_batch("PRAGMA foreign_keys = ON;")
        .expect("enable foreign keys");
    conn.execute_batch(SESSION_SCHEMA).expect("session schema");
    conn.execute_batch(db::SCHEMA).expect("bookmarks schema");
    let db: Db = Arc::new(Mutex::new(conn));

    let cookie_secure = std::env::var("COOKIE_SECURE")
        .map(|v| v.eq_ignore_ascii_case("true"))
        .unwrap_or(false);

    let state = Arc::new(AppState {
        db,
        admin_password: std::env::var("BOOKMARKS_PASSWORD").ok().filter(|s| !s.is_empty()),
        api_key: std::env::var("BOOKMARKS_API_KEY").ok().filter(|s| !s.is_empty()),
        cookie_secure,
    });

    let api_authed = Router::new()
        .route("/api/links", post(api_create_link))
        .route_layer(middleware::from_fn_with_state(state.clone(), require_api_key));

    let api_open = Router::new()
        .route("/api/categories", get(api_list_categories))
        .route("/api/links", get(api_list_links));

    let app = Router::new()
        .route("/", get(index_handler))
        .route("/login", get(login_get).post(login_post))
        .route("/logout", get(logout_handler))
        .route("/admin", get(admin_handler))
        .route("/admin/categories", post(admin_add_category))
        .route("/admin/categories/{short_id}/delete", post(admin_delete_category))
        .route("/admin/links", post(admin_add_link))
        .route("/admin/links/{short_id}/delete", post(admin_delete_link))
        .route("/static/{*path}", get(static_handler))
        .merge(api_authed)
        .merge(api_open)
        .merge(andromeda_darkmatter_css::router::<Arc<AppState>>())
        .with_state(state);

    let host = std::env::var("HOST").unwrap_or_else(|_| "127.0.0.1".to_string());
    let port: u16 = std::env::var("PORT")
        .ok()
        .and_then(|v| v.parse().ok())
        .unwrap_or(3000);
    let addr = format!("{host}:{port}");
    let listener = tokio::net::TcpListener::bind(&addr)
        .await
        .unwrap_or_else(|_| panic!("Failed to bind to {addr}"));

    tracing::info!("Bookmarks server running on http://{host}:{port}");
    axum::serve(listener, app).await.unwrap();
}
