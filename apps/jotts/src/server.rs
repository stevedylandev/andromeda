use askama::Template;
use askama_web::WebTemplate;
use axum::{
    extract::{Form, Path, Query, Request, State},
    http::{HeaderValue, StatusCode},
    middleware::{self, Next},
    response::{Html, IntoResponse, Redirect, Response},
    routing::{get, post},
    Json, Router,
};
use pulldown_cmark::{Options, Parser, html}; use rust_embed::Embed;
use std::sync::Arc;

use crate::auth;
use crate::db::{self, Db, DbError, Note, NoteInput};

fn redirect_with_cookie(target: &str, cookie: String) -> Response {
    let mut resp = Redirect::to(target).into_response();
    resp.headers_mut().insert(
        axum::http::header::SET_COOKIE,
        HeaderValue::from_str(&cookie).unwrap(),
    );
    resp
}

#[derive(Clone)]
pub struct AppState {
    pub db: Db,
    pub app_password: String,
    pub api_key: Option<String>,
    pub cookie_secure: bool,
}

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

// --- API key middleware ---

async fn api_key_guard(
    State(state): State<Arc<AppState>>,
    req: Request,
    next: Next,
) -> Response {
    let expected = match &state.api_key {
        Some(k) if !k.is_empty() => k.clone(),
        _ => {
            return (StatusCode::FORBIDDEN, "API key not configured on server").into_response();
        }
    };

    let provided = req
        .headers()
        .get("x-api-key")
        .and_then(|v| v.to_str().ok())
        .unwrap_or("");

    if !andromeda_auth::verify_api_key(provided, &expected) {
        return (StatusCode::UNAUTHORIZED, "Invalid API key").into_response();
    }

    next.run(req).await
}

// --- JSON API handlers ---

async fn api_list_notes(State(state): State<Arc<AppState>>) -> Result<Response, DbError> {
    Ok(Json(db::get_all_notes(&state.db)?).into_response())
}

async fn api_get_note(
    State(state): State<Arc<AppState>>,
    Path(short_id): Path<String>,
) -> Result<Response, DbError> {
    Ok(match db::get_note_by_short_id(&state.db, &short_id)? {
        Some(note) => Json(note).into_response(),
        None => StatusCode::NOT_FOUND.into_response(),
    })
}

async fn api_create_note(
    State(state): State<Arc<AppState>>,
    Json(body): Json<NoteInput>,
) -> Result<Response, DbError> {
    let title = body.title.trim();
    if title.is_empty() {
        return Ok((StatusCode::BAD_REQUEST, "title required").into_response());
    }
    let note = db::create_note(&state.db, title, &body.content)?;
    Ok((StatusCode::CREATED, Json(note)).into_response())
}

async fn api_update_note(
    State(state): State<Arc<AppState>>,
    Path(short_id): Path<String>,
    Json(body): Json<NoteInput>,
) -> Result<Response, DbError> {
    let title = body.title.trim();
    if title.is_empty() {
        return Ok((StatusCode::BAD_REQUEST, "title required").into_response());
    }
    Ok(
        match db::update_note_by_short_id(&state.db, &short_id, title, &body.content)? {
            Some(note) => Json(note).into_response(),
            None => StatusCode::NOT_FOUND.into_response(),
        },
    )
}

async fn api_delete_note(
    State(state): State<Arc<AppState>>,
    Path(short_id): Path<String>,
) -> Result<Response, DbError> {
    Ok(match db::delete_note_by_short_id(&state.db, &short_id)? {
        true => StatusCode::NO_CONTENT.into_response(),
        false => StatusCode::NOT_FOUND.into_response(),
    })
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
    let expires_at = andromeda_auth::datetime::expiry_datetime_string(7 * 24 * 3600);

    if let Err(e) = db::insert_session(&state.db, &token, &expires_at) {
        tracing::error!("Failed to create session: {}", e);
        return Redirect::to("/login?error=Server+error").into_response();
    }

    redirect_with_cookie("/", auth::build_session_cookie(&token, state.cookie_secure))
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

    redirect_with_cookie("/login", auth::clear_session_cookie())
}

// --- Note handlers ---

async fn get_index(
    _session: auth::AuthSession,
    State(state): State<Arc<AppState>>,
) -> Result<Response, DbError> {
    let notes = db::get_all_notes(&state.db)?;
    Ok(WebTemplate(IndexTemplate { notes }).into_response())
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
    Form(form): Form<NoteInput>,
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
) -> Result<Response, DbError> {
    Ok(match db::get_note_by_short_id(&state.db, &short_id)? {
        Some(note) => {
            let rendered_content = render_markdown(&note.content);
            WebTemplate(ViewTemplate {
                note,
                rendered_content,
            })
            .into_response()
        }
        None => (StatusCode::NOT_FOUND, Html("Note not found".to_string())).into_response(),
    })
}

async fn get_edit_note(
    _session: auth::AuthSession,
    State(state): State<Arc<AppState>>,
    Path(short_id): Path<String>,
    Query(q): Query<FlashQuery>,
) -> Result<Response, DbError> {
    Ok(match db::get_note_by_short_id(&state.db, &short_id)? {
        Some(note) => WebTemplate(EditTemplate {
            note,
            error: q.error,
        })
        .into_response(),
        None => (StatusCode::NOT_FOUND, Html("Note not found".to_string())).into_response(),
    })
}

async fn post_update_note(
    _session: auth::AuthSession,
    State(state): State<Arc<AppState>>,
    Path(short_id): Path<String>,
    Form(form): Form<NoteInput>,
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

    let api_key = std::env::var("JOTTS_API_KEY")
        .ok()
        .filter(|k| !k.is_empty());
    if api_key.is_none() {
        tracing::info!("JOTTS_API_KEY not set, /api/* will return 403");
    }

    let state = Arc::new(AppState {
        db,
        app_password,
        api_key,
        cookie_secure,
    });

    let api_router = Router::new()
        .route("/api/notes", get(api_list_notes).post(api_create_note))
        .route(
            "/api/notes/{short_id}",
            get(api_get_note)
                .put(api_update_note)
                .delete(api_delete_note),
        )
        .route_layer(middleware::from_fn_with_state(
            state.clone(),
            api_key_guard,
        ));

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
        .route("/static/{*path}", get(serve_static))
        .merge(api_router)
        .merge(andromeda_darkmatter_css::router::<Arc<AppState>>())
        .with_state(state);

    let addr = format!("{}:{}", host, port);
    tracing::info!("Listening on http://{}", addr);

    let listener = tokio::net::TcpListener::bind(&addr).await.unwrap();
    axum::serve(listener, app).await.unwrap();
}
