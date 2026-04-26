mod auth;
mod db;
mod google_books;

use std::sync::{Arc, Mutex};

use andromeda_db::{
    session::{prune_expired_sessions, SESSION_SCHEMA},
    Db,
};
use askama::Template;
use axum::{
    extract::{Path, Query, State},
    http::{header, HeaderMap, StatusCode},
    response::{Html, IntoResponse, Json, Redirect, Response},
    routing::{get, post},
    Form, Router,
};
use rusqlite::Connection;
use rust_embed::Embed;
use serde::Deserialize;

use crate::db::{Book, BookStatus, CategoryLabels, NewBook};

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum DisplayMode {
    Inline,
    Nav,
}

impl DisplayMode {
    fn parse(s: &str) -> Self {
        match s.to_ascii_lowercase().as_str() {
            "nav" => DisplayMode::Nav,
            _ => DisplayMode::Inline,
        }
    }
}

#[derive(Embed)]
#[folder = "static/"]
struct Static;

pub struct AppState {
    pub db: Db,
    pub admin_password: Option<String>,
    pub google_books_api_key: Option<String>,
    pub cookie_secure: bool,
    pub base_url: String,
    pub display_mode: DisplayMode,
}

// ── Templates ────────────────────────────────────────────────────────────

struct BookView {
    title: String,
    authors: String,
    cover_url: Option<String>,
    notes: Option<String>,
}

struct SectionView {
    label: String,
    books: Vec<BookView>,
}

struct NavCategory {
    slug: &'static str,
    label: String,
    active: bool,
}

#[derive(Template)]
#[template(path = "index.html")]
struct IndexTemplate {
    base_url: String,
    sections: Vec<SectionView>,
    nav_mode: bool,
    nav_categories: Vec<NavCategory>,
}

#[derive(Template)]
#[template(path = "login.html")]
struct LoginTemplate {
    error: Option<String>,
}

struct AdminBookRow {
    id: i64,
    title: String,
    authors: String,
    isbn: Option<String>,
    cover_url: Option<String>,
    notes: Option<String>,
    status: String,
}

#[derive(Template)]
#[template(path = "admin.html")]
struct AdminTemplate {
    success: Option<String>,
    error: Option<String>,
    books: Vec<AdminBookRow>,
    labels: CategoryLabels,
    library_query: String,
    library_results: Vec<AdminBookRow>,
    library_searched: bool,
}

fn make_book_view(b: Book) -> BookView {
    BookView {
        title: b.title,
        authors: b.authors,
        cover_url: b.cover_url,
        notes: b.notes,
    }
}

#[derive(Deserialize, Default)]
struct IndexQuery {
    category: Option<String>,
}

async fn index_handler(
    State(state): State<Arc<AppState>>,
    Query(q): Query<IndexQuery>,
) -> Response {
    let all_books = db::list_books(&state.db, None).unwrap_or_default();
    let labels = db::get_category_labels(&state.db).unwrap_or_default();

    let section_defs: &[(&'static str, BookStatus)] = &[
        ("reading", BookStatus::Reading),
        ("read", BookStatus::Read),
        ("want", BookStatus::Want),
    ];

    let make_section = |status: BookStatus| -> SectionView {
        let books = all_books
            .iter()
            .filter(|b| b.status == status.as_str())
            .map(|b| make_book_view(b.clone()))
            .collect();
        SectionView {
            label: labels.label_for(status).to_string(),
            books,
        }
    };

    let nav_mode = matches!(state.display_mode, DisplayMode::Nav);

    let (sections, nav_categories) = if nav_mode {
        let selected_slug = q
            .category
            .as_deref()
            .and_then(|s| {
                section_defs
                    .iter()
                    .find(|(slug, _)| *slug == s)
                    .map(|(slug, _)| *slug)
            })
            .unwrap_or(section_defs[0].0);

        let nav = section_defs
            .iter()
            .map(|(slug, status)| NavCategory {
                slug,
                label: labels.label_for(*status).to_string(),
                active: *slug == selected_slug,
            })
            .collect();

        let (_, status) = section_defs
            .iter()
            .find(|(slug, _)| *slug == selected_slug)
            .copied()
            .unwrap();
        (vec![make_section(status)], nav)
    } else {
        let secs = section_defs
            .iter()
            .filter_map(|(_, status)| {
                let s = make_section(*status);
                if s.books.is_empty() { None } else { Some(s) }
            })
            .collect();
        (secs, Vec::new())
    };

    Html(
        IndexTemplate {
            base_url: state.base_url.clone(),
            sections,
            nav_mode,
            nav_categories,
        }
        .render()
        .unwrap(),
    )
    .into_response()
}

async fn static_handler(Path(path): Path<String>) -> Response {
    match Static::get(&path) {
        Some(file) => {
            let mime = mime_guess::from_path(&path).first_or_octet_stream();
            ([(header::CONTENT_TYPE, mime.as_ref())], file.data.to_vec()).into_response()
        }
        None => StatusCode::NOT_FOUND.into_response(),
    }
}

// ── Admin ────────────────────────────────────────────────────────────────

#[derive(Deserialize, Default)]
struct FlashQuery {
    error: Option<String>,
}

#[derive(Deserialize)]
struct LoginForm {
    password: String,
}

async fn login_get_handler(Query(q): Query<FlashQuery>) -> Response {
    Html(LoginTemplate { error: q.error }.render().unwrap()).into_response()
}

async fn login_post_handler(
    State(state): State<Arc<AppState>>,
    Form(form): Form<LoginForm>,
) -> Response {
    let admin_password = match &state.admin_password {
        Some(p) => p,
        None => {
            return Redirect::to("/admin/login?error=No+admin+password+configured").into_response();
        }
    };
    if !auth::verify_password(&form.password, admin_password) {
        return Redirect::to("/admin/login?error=Invalid+password").into_response();
    }

    let token = auth::generate_session_token();
    if let Err(e) = auth::create_session(&state.db, &token) {
        tracing::error!("failed to create session: {e}");
        return Redirect::to("/admin/login?error=Session+error").into_response();
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
    let mut resp = Redirect::to("/admin/login").into_response();
    resp.headers_mut().insert(
        header::SET_COOKIE,
        auth::clear_session_cookie().parse().unwrap(),
    );
    resp
}

fn book_to_row(b: Book) -> AdminBookRow {
    AdminBookRow {
        id: b.id,
        title: b.title,
        authors: b.authors,
        isbn: b.isbn,
        cover_url: b.cover_url,
        notes: b.notes,
        status: b.status,
    }
}

#[derive(Deserialize, Default)]
struct AdminQuery {
    error: Option<String>,
    success: Option<String>,
    q: Option<String>,
}

async fn admin_handler(
    _session: auth::AuthSession,
    State(state): State<Arc<AppState>>,
    Query(q): Query<AdminQuery>,
) -> Response {
    let books = db::list_books(&state.db, None)
        .unwrap_or_default()
        .into_iter()
        .map(book_to_row)
        .collect();
    let labels = db::get_category_labels(&state.db).unwrap_or_default();

    let library_query = q.q.unwrap_or_default();
    let library_searched = !library_query.trim().is_empty();
    let library_results = if library_searched {
        db::search_books(&state.db, &library_query)
            .unwrap_or_default()
            .into_iter()
            .map(book_to_row)
            .collect()
    } else {
        Vec::new()
    };

    Html(
        AdminTemplate {
            success: q.success,
            error: q.error,
            books,
            labels,
            library_query,
            library_results,
            library_searched,
        }
        .render()
        .unwrap(),
    )
    .into_response()
}

#[derive(Deserialize)]
struct CategoryLabelsForm {
    reading: String,
    read: String,
    want: String,
}

async fn admin_update_labels(
    _session: auth::AuthSession,
    State(state): State<Arc<AppState>>,
    Form(form): Form<CategoryLabelsForm>,
) -> Response {
    let reading = form.reading.trim();
    let read = form.read.trim();
    let want = form.want.trim();
    if reading.is_empty() || read.is_empty() || want.is_empty() {
        return Redirect::to("/admin?error=Labels+cannot+be+empty").into_response();
    }
    if let Err(e) = db::set_setting(&state.db, "category_label.reading", reading)
        .and_then(|_| db::set_setting(&state.db, "category_label.read", read))
        .and_then(|_| db::set_setting(&state.db, "category_label.want", want))
    {
        tracing::error!("save labels: {e}");
        return Redirect::to("/admin?error=Failed+to+save+labels").into_response();
    }
    Redirect::to("/admin?success=Labels+updated").into_response()
}

#[derive(Deserialize)]
struct SearchQuery {
    q: String,
}

async fn admin_search_handler(
    _session: auth::AuthSession,
    State(state): State<Arc<AppState>>,
    Query(q): Query<SearchQuery>,
) -> Response {
    match google_books::search(&q.q, state.google_books_api_key.as_deref()).await {
        Ok(hits) => Json(hits).into_response(),
        Err(e) => {
            tracing::warn!("google books search failed: {e}");
            (StatusCode::BAD_GATEWAY, Json(serde_json::json!({ "error": e })))
                .into_response()
        }
    }
}

#[derive(Deserialize)]
struct AddBookForm {
    google_id: Option<String>,
    title: String,
    authors: String,
    isbn: Option<String>,
    cover_url: Option<String>,
    status: String,
}

async fn admin_add_book(
    _session: auth::AuthSession,
    State(state): State<Arc<AppState>>,
    Form(form): Form<AddBookForm>,
) -> Response {
    let Some(status) = BookStatus::parse(&form.status) else {
        return Redirect::to("/admin?error=Invalid+status").into_response();
    };
    let new_book = NewBook {
        google_id: form.google_id.filter(|s| !s.is_empty()),
        title: form.title,
        authors: form.authors,
        isbn: form.isbn.filter(|s| !s.is_empty()),
        cover_url: form.cover_url.filter(|s| !s.is_empty()),
        notes: None,
        status,
    };
    match db::insert_book(&state.db, &new_book) {
        Ok(_) => Redirect::to("/admin?success=Book+added").into_response(),
        Err(e) => {
            tracing::error!("insert book: {e}");
            Redirect::to("/admin?error=Failed+to+add+book").into_response()
        }
    }
}

#[derive(Deserialize)]
struct UpdateStatusForm {
    status: String,
}

async fn admin_update_status(
    _session: auth::AuthSession,
    State(state): State<Arc<AppState>>,
    Path(id): Path<i64>,
    Form(form): Form<UpdateStatusForm>,
) -> Response {
    let Some(status) = BookStatus::parse(&form.status) else {
        return Redirect::to("/admin?error=Invalid+status").into_response();
    };
    let _ = db::update_book_status(&state.db, id, status);
    Redirect::to("/admin?success=Status+updated").into_response()
}

#[derive(Deserialize)]
struct UpdateNotesForm {
    notes: String,
}

async fn admin_update_notes(
    _session: auth::AuthSession,
    State(state): State<Arc<AppState>>,
    Path(id): Path<i64>,
    Form(form): Form<UpdateNotesForm>,
) -> Response {
    let trimmed = form.notes.trim();
    let notes = if trimmed.is_empty() { None } else { Some(trimmed) };
    let _ = db::update_book_notes(&state.db, id, notes);
    Redirect::to("/admin?success=Notes+saved").into_response()
}

async fn admin_delete_book(
    _session: auth::AuthSession,
    State(state): State<Arc<AppState>>,
    Path(id): Path<i64>,
) -> Response {
    let _ = db::delete_book(&state.db, id);
    Redirect::to("/admin?success=Book+removed").into_response()
}

// ── JSON API ─────────────────────────────────────────────────────────────

#[derive(Deserialize)]
struct ListBooksQuery {
    status: Option<String>,
}

async fn api_list_books(
    State(state): State<Arc<AppState>>,
    Query(q): Query<ListBooksQuery>,
) -> Response {
    let status = match q.status.as_deref() {
        None | Some("") | Some("all") => None,
        Some(s) => match BookStatus::parse(s) {
            Some(st) => Some(st),
            None => {
                return (
                    StatusCode::BAD_REQUEST,
                    Json(serde_json::json!({ "error": "invalid status" })),
                )
                    .into_response();
            }
        },
    };
    match db::list_books(&state.db, status) {
        Ok(books) => Json(books).into_response(),
        Err(e) => {
            tracing::error!("list books: {e}");
            StatusCode::INTERNAL_SERVER_ERROR.into_response()
        }
    }
}

async fn api_get_book(
    State(state): State<Arc<AppState>>,
    Path(id): Path<i64>,
) -> Response {
    match db::get_book(&state.db, id) {
        Ok(Some(book)) => Json(book).into_response(),
        Ok(None) => (StatusCode::NOT_FOUND, Json(serde_json::json!({ "error": "not found" })))
            .into_response(),
        Err(e) => {
            tracing::error!("get book: {e}");
            StatusCode::INTERNAL_SERVER_ERROR.into_response()
        }
    }
}

// ── main ─────────────────────────────────────────────────────────────────

#[tokio::main]
async fn main() {
    dotenvy::dotenv().ok();
    tracing_subscriber::fmt()
        .with_env_filter(
            tracing_subscriber::EnvFilter::try_from_default_env()
                .unwrap_or_else(|_| tracing_subscriber::EnvFilter::new("info,library=info")),
        )
        .init();

    let db_path =
        std::env::var("LIBRARY_DB_PATH").unwrap_or_else(|_| "library.sqlite".to_string());
    let conn = Connection::open(&db_path).expect("open sqlite");
    conn.execute_batch(SESSION_SCHEMA).expect("session schema");
    conn.execute_batch(db::BOOKS_SCHEMA).expect("books schema");
    let db: Db = Arc::new(Mutex::new(conn));

    let cookie_secure = std::env::var("COOKIE_SECURE")
        .map(|v| v.eq_ignore_ascii_case("true"))
        .unwrap_or(false);
    let base_url =
        std::env::var("BASE_URL").unwrap_or_else(|_| "http://localhost:3000".to_string());

    let google_books_api_key = std::env::var("GOOGLE_BOOKS_API_KEY")
        .ok()
        .filter(|s| !s.is_empty());

    let display_mode = std::env::var("LIBRARY_DISPLAY_MODE")
        .map(|v| DisplayMode::parse(&v))
        .unwrap_or(DisplayMode::Inline);

    let state = Arc::new(AppState {
        db,
        admin_password: std::env::var("ADMIN_PASSWORD").ok(),
        google_books_api_key,
        cookie_secure,
        base_url,
        display_mode,
    });

    let admin_router = Router::new()
        .route("/admin", get(admin_handler))
        .route(
            "/admin/login",
            get(login_get_handler).post(login_post_handler),
        )
        .route("/admin/logout", get(logout_handler))
        .route("/admin/search", get(admin_search_handler))
        .route("/admin/categories/labels", post(admin_update_labels))
        .route("/admin/add", post(admin_add_book))
        .route("/admin/books/{id}/status", post(admin_update_status))
        .route("/admin/books/{id}/notes", post(admin_update_notes))
        .route("/admin/books/{id}/delete", post(admin_delete_book));

    let api_router = Router::new()
        .route("/api/books", get(api_list_books))
        .route("/api/books/{id}", get(api_get_book));

    let app = Router::new()
        .route("/", get(index_handler))
        .route("/static/{*path}", get(static_handler))
        .merge(admin_router)
        .merge(api_router)
        .merge(andromeda_darkmatter_css::router::<Arc<AppState>>())
        .with_state(state);

    let host = std::env::var("HOST").unwrap_or_else(|_| "0.0.0.0".to_string());
    let port: u16 = std::env::var("PORT")
        .ok()
        .and_then(|v| v.parse().ok())
        .unwrap_or(3000);
    let addr = format!("{host}:{port}");
    let listener = tokio::net::TcpListener::bind(&addr)
        .await
        .unwrap_or_else(|_| panic!("Failed to bind to {addr}"));

    tracing::info!("Library server running on http://{host}:{port}");
    axum::serve(listener, app).await.unwrap();
}

