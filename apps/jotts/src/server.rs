use askama::Template;
use askama_web::WebTemplate;
use axum::{
    extract::{Form, Path, Query, State},
    http::{HeaderValue, StatusCode},
    response::{Html, IntoResponse, Redirect, Response},
    routing::{get, post},
    Router,
};
use pulldown_cmark::{Options, Parser, html}; use rust_embed::Embed;
use std::sync::Arc;

use crate::auth;
use crate::db::{self, Db, Note};

#[derive(Clone)]
pub struct AppState {
    pub db: Db,
    pub app_password: String,
    pub cookie_secure: bool,
}

#[derive(Embed)]
#[folder = "assets/"]
struct Assets;

#[derive(Embed)]
#[folder = "static/"]
struct Static;

// --- Templates ---

#[derive(Template)]
#[template(path = "base.html")]
struct BaseTemplate;

#[derive(Template)]
#[template(path = "login.html")]
struct LoginTemplate {
    error: Option<String>,
}

#[derive(Template)]
#[template(path = "index.html")]
struct IndexTemplate {
    notes: Vec<Note>,
}

#[derive(Template)]
#[template(path = "view.html")]
struct ViewTemplate {
    note: Note,
    rendered_content: String,
}

#[derive(Template)]
#[template(path = "new.html")]
struct NewTemplate {
    error: Option<String>,
}

#[derive(Template)]
#[template(path = "edit.html")]
struct EditTemplate {
    note: Note,
    error: Option<String>,
}

// --- Query/Form structs ---

#[derive(serde::Deserialize, Default)]
pub struct FlashQuery {
    pub error: Option<String>,
}

#[derive(serde::Deserialize)]
struct LoginForm {
    password: String,
}

#[derive(serde::Deserialize)]
struct NoteForm {
    title: String,
    content: String,
}

// --- Static file handlers ---

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

async fn serve_asset(Path(path): Path<String>) -> Response {
    match Assets::get(&path) {
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
        return Redirect::to("/login?error=Invalid+password").into_response();
    }

    let token = auth::generate_session_token();

    // Session expires in 7 days
    // We need to compute a datetime 7 days from now
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
            y,
            m,
            d,
            tod / 3600,
            (tod % 3600) / 60,
            tod % 60
        )
    };

    if let Err(e) = db::insert_session(&state.db, &token, &expires_at) {
        tracing::error!("Failed to create session: {}", e);
        return Redirect::to("/login?error=Server+error").into_response();
    }

    let cookie = auth::build_session_cookie(&token, state.cookie_secure);
    let mut resp = Redirect::to("/").into_response();
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
    let mut resp = Redirect::to("/login").into_response();
    resp.headers_mut().insert(
        axum::http::header::SET_COOKIE,
        HeaderValue::from_str(&cookie).unwrap(),
    );
    resp
}

// --- Note handlers ---

async fn get_index(
    _session: auth::AuthSession,
    State(state): State<Arc<AppState>>,
) -> Response {
    match db::get_all_notes(&state.db) {
        Ok(notes) => WebTemplate(IndexTemplate { notes }).into_response(),
        Err(e) => {
            tracing::error!("Failed to list notes: {}", e);
            (StatusCode::INTERNAL_SERVER_ERROR, Html("Server error".to_string())).into_response()
        }
    }
}

async fn get_new_note(
    _session: auth::AuthSession,
    Query(q): Query<FlashQuery>,
) -> Response {
    WebTemplate(NewTemplate { error: q.error }).into_response()
}

async fn post_create_note(
    _session: auth::AuthSession,
    State(state): State<Arc<AppState>>,
    Form(form): Form<NoteForm>,
) -> Response {
    let title = form.title.trim();
    if title.is_empty() {
        return Redirect::to("/notes/new?error=Title+is+required").into_response();
    }

    match db::create_note(&state.db, title, &form.content) {
        Ok(note) => Redirect::to(&format!("/notes/{}", note.short_id)).into_response(),
        Err(e) => {
            tracing::error!("Failed to create note: {}", e);
            Redirect::to("/notes/new?error=Failed+to+create+note").into_response()
        }
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

async fn get_view_note(
    _session: auth::AuthSession,
    State(state): State<Arc<AppState>>,
    Path(short_id): Path<String>,
) -> Response {
    match db::get_note_by_short_id(&state.db, &short_id) {
        Ok(Some(note)) => {
            let rendered_content = render_markdown(&note.content);
            WebTemplate(ViewTemplate {
                note,
                rendered_content,
            })
            .into_response()
        }
        Ok(None) => (StatusCode::NOT_FOUND, Html("Note not found".to_string())).into_response(),
        Err(e) => {
            tracing::error!("Failed to get note: {}", e);
            (StatusCode::INTERNAL_SERVER_ERROR, Html("Server error".to_string())).into_response()
        }
    }
}

async fn get_edit_note(
    _session: auth::AuthSession,
    State(state): State<Arc<AppState>>,
    Path(short_id): Path<String>,
    Query(q): Query<FlashQuery>,
) -> Response {
    match db::get_note_by_short_id(&state.db, &short_id) {
        Ok(Some(note)) => WebTemplate(EditTemplate {
            note,
            error: q.error,
        })
        .into_response(),
        Ok(None) => (StatusCode::NOT_FOUND, Html("Note not found".to_string())).into_response(),
        Err(e) => {
            tracing::error!("Failed to get note: {}", e);
            (StatusCode::INTERNAL_SERVER_ERROR, Html("Server error".to_string())).into_response()
        }
    }
}

async fn post_update_note(
    _session: auth::AuthSession,
    State(state): State<Arc<AppState>>,
    Path(short_id): Path<String>,
    Form(form): Form<NoteForm>,
) -> Response {
    let title = form.title.trim();
    if title.is_empty() {
        return Redirect::to(&format!("/notes/{}/edit?error=Title+is+required", short_id))
            .into_response();
    }

    match db::update_note_by_short_id(&state.db, &short_id, title, &form.content) {
        Ok(Some(_)) => Redirect::to(&format!("/notes/{}", short_id)).into_response(),
        Ok(None) => (StatusCode::NOT_FOUND, Html("Note not found".to_string())).into_response(),
        Err(e) => {
            tracing::error!("Failed to update note: {}", e);
            Redirect::to(&format!(
                "/notes/{}/edit?error=Failed+to+update+note",
                short_id
            ))
            .into_response()
        }
    }
}

async fn post_delete_note(
    _session: auth::AuthSession,
    State(state): State<Arc<AppState>>,
    Path(short_id): Path<String>,
) -> Response {
    match db::delete_note_by_short_id(&state.db, &short_id) {
        Ok(_) => Redirect::to("/").into_response(),
        Err(e) => {
            tracing::error!("Failed to delete note: {}", e);
            Redirect::to("/").into_response()
        }
    }
}

// --- Date helper (same algorithm as auth.rs) ---

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

    // Prune expired sessions on startup
    if let Err(e) = db::prune_expired_sessions(&db) {
        tracing::warn!("Failed to prune sessions: {}", e);
    }

    let app_password = std::env::var("JOTTS_PASSWORD").unwrap_or_else(|_| {
        tracing::warn!("JOTTS_PASSWORD not set, using default 'changeme'");
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
        .route("/login", get(get_login).post(post_login))
        .route("/logout", get(get_logout))
        // Protected routes
        .route("/", get(get_index))
        .route("/notes/new", get(get_new_note))
        .route("/notes", post(post_create_note))
        .route("/notes/{short_id}", get(get_view_note))
        .route("/notes/{short_id}/edit", get(get_edit_note))
        .route("/notes/{short_id}", post(post_update_note))
        .route("/notes/{short_id}/delete", post(post_delete_note))
        // Static assets
        .route("/assets/{*path}", get(serve_asset))
        .route("/static/{*path}", get(serve_static))
        .with_state(state);

    let addr = format!("{}:{}", host, port);
    tracing::info!("Listening on http://{}", addr);

    let listener = tokio::net::TcpListener::bind(&addr).await.unwrap();
    axum::serve(listener, app).await.unwrap();
}
