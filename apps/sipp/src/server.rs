use askama::Template;
use askama_web::WebTemplate;
use subtle::ConstantTimeEq;
use axum::{
    Form, Json, Router,
    extract::{Path, Request, State},
    http::{HeaderMap, StatusCode, header},
    middleware::{self, Next},
    response::{Html, IntoResponse, Redirect, Response},
    routing::{delete, get, post, put},
};
use rust_embed::Embed;
use serde::Deserialize;
use crate::db::{self, Db, Snippet};
use crate::highlight::Highlighter;
use std::collections::HashSet;
use std::sync::Arc;

#[derive(Embed)]
#[folder = "assets/"]
struct Assets;

#[derive(Embed)]
#[folder = "static/"]
struct Static;

#[derive(Clone)]
struct ServerConfig {
    api_key: Option<String>,
    auth_endpoints: HashSet<String>,
    max_content_size: usize,
}

impl ServerConfig {
    fn from_env() -> Self {
        let api_key = std::env::var("SIPP_API_KEY").ok();
        let auth_endpoints = match std::env::var("SIPP_AUTH_ENDPOINTS") {
            Ok(val) if val.trim().eq_ignore_ascii_case("none") => HashSet::new(),
            Ok(val) => val.split(',').map(|s| s.trim().to_lowercase()).collect(),
            Err(_) => ["api_delete", "api_list", "api_update"].iter().map(|s| s.to_string()).collect(),
        };
        let max_content_size = std::env::var("SIPP_MAX_CONTENT_SIZE")
            .ok()
            .and_then(|v| v.parse().ok())
            .unwrap_or(512_000);
        ServerConfig { api_key, auth_endpoints, max_content_size }
    }

    fn requires_auth(&self, name: &str) -> bool {
        self.auth_endpoints.contains("all") || self.auth_endpoints.contains(name)
    }
}

#[derive(Clone)]
struct AppState {
    db: Db,
    highlighter: Arc<Highlighter>,
    server_config: ServerConfig,
}

#[derive(Template)]
#[template(path = "index.html")]
struct IndexTemplate;

#[derive(Template)]
#[template(path = "admin.html")]
struct AdminTemplate;

#[derive(Template)]
#[template(path = "snippet.html")]
struct SnippetTemplate {
    name: String,
    content: String,
    highlighted_content: String,
}

#[derive(Deserialize)]
struct CreateSnippetForm {
    name: String,
    content: String,
}

async fn index() -> WebTemplate<IndexTemplate> {
    WebTemplate(IndexTemplate)
}

async fn admin() -> WebTemplate<AdminTemplate> {
    WebTemplate(AdminTemplate)
}

fn is_cli_user_agent(headers: &HeaderMap) -> bool {
    headers
        .get(header::USER_AGENT)
        .and_then(|v| v.to_str().ok())
        .map(|ua| {
            let ua = ua.to_lowercase();
            ua.starts_with("curl/") || ua.starts_with("wget/") || ua.starts_with("httpie/")
        })
        .unwrap_or(false)
}

async fn view_snippet(
    State(state): State<AppState>,
    Path(short_id): Path<String>,
    headers: HeaderMap,
) -> Result<Response, (StatusCode, Html<String>)> {
    match db::get_snippet_by_short_id(&state.db, &short_id) {
        Ok(Some(snippet)) => {
            if is_cli_user_agent(&headers) {
                Ok((
                    [(header::CONTENT_TYPE, "text/plain; charset=utf-8")],
                    snippet.content,
                )
                    .into_response())
            } else {
                let highlighted_content =
                    state.highlighter.highlight(&snippet.name, &snippet.content);
                Ok(WebTemplate(SnippetTemplate {
                    name: snippet.name,
                    content: snippet.content,
                    highlighted_content,
                })
                .into_response())
            }
        }
        Ok(None) => Err((
            StatusCode::NOT_FOUND,
            Html("<h1>Snippet not found</h1>".to_string()),
        )),
        Err(_) => Err((
            StatusCode::INTERNAL_SERVER_ERROR,
            Html("<h1>Internal server error</h1>".to_string()),
        )),
    }
}

async fn create_snippet(
    State(state): State<AppState>,
    Form(form): Form<CreateSnippetForm>,
) -> Result<Redirect, (StatusCode, Html<String>)> {
    if form.content.len() > state.server_config.max_content_size {
        return Err((
            StatusCode::PAYLOAD_TOO_LARGE,
            Html(format!(
                "<h1>Content too large</h1><p>Maximum size is {} bytes</p>",
                state.server_config.max_content_size
            )),
        ));
    }
    match db::create_snippet(&state.db, &form.name, &form.content) {
        Ok(snippet) => Ok(Redirect::to(&format!("/s/{}", snippet.short_id))),
        Err(_) => Err((
            StatusCode::INTERNAL_SERVER_ERROR,
            Html("<h1>Internal server error</h1>".to_string()),
        )),
    }
}

async fn require_api_key(
    State(state): State<AppState>,
    headers: HeaderMap,
    request: Request,
    next: Next,
) -> Result<Response, (StatusCode, Json<serde_json::Value>)> {
    let server_key = match &state.server_config.api_key {
        Some(k) => k,
        None => return Err((
            StatusCode::FORBIDDEN,
            Json(serde_json::json!({"error": "No API key configured on server"})),
        )),
    };
    let provided = headers
        .get("x-api-key")
        .and_then(|v| v.to_str().ok());
    match provided {
        Some(k) if k.as_bytes().ct_eq(server_key.as_bytes()).into() => Ok(next.run(request).await),
        _ => Err((
            StatusCode::UNAUTHORIZED,
            Json(serde_json::json!({"error": "Invalid or missing API key"})),
        )),
    }
}

async fn api_list_snippets(
    State(state): State<AppState>,
) -> Result<Json<Vec<Snippet>>, (StatusCode, Json<serde_json::Value>)> {
    match db::get_all_snippets(&state.db) {
        Ok(snippets) => Ok(Json(snippets)),
        Err(_) => Err((StatusCode::INTERNAL_SERVER_ERROR, Json(serde_json::json!({"error": "Internal server error"})))),
    }
}

async fn api_get_snippet(
    State(state): State<AppState>,
    Path(short_id): Path<String>,
) -> Result<Json<Snippet>, (StatusCode, Json<serde_json::Value>)> {
    match db::get_snippet_by_short_id(&state.db, &short_id) {
        Ok(Some(snippet)) => Ok(Json(snippet)),
        Ok(None) => Err((StatusCode::NOT_FOUND, Json(serde_json::json!({"error": "Snippet not found"})))),
        Err(_) => Err((StatusCode::INTERNAL_SERVER_ERROR, Json(serde_json::json!({"error": "Internal server error"})))),
    }
}

#[derive(Deserialize)]
struct ApiCreateSnippet {
    name: String,
    content: String,
}

async fn api_create_snippet(
    State(state): State<AppState>,
    Json(body): Json<ApiCreateSnippet>,
) -> Result<(StatusCode, Json<Snippet>), (StatusCode, Json<serde_json::Value>)> {
    if body.content.len() > state.server_config.max_content_size {
        return Err((
            StatusCode::PAYLOAD_TOO_LARGE,
            Json(serde_json::json!({
                "error": format!("Content too large. Maximum size is {} bytes", state.server_config.max_content_size)
            })),
        ));
    }
    match db::create_snippet(&state.db, &body.name, &body.content) {
        Ok(snippet) => Ok((StatusCode::CREATED, Json(snippet))),
        Err(_) => Err((StatusCode::INTERNAL_SERVER_ERROR, Json(serde_json::json!({"error": "Internal server error"})))),
    }
}

async fn api_delete_snippet(
    State(state): State<AppState>,
    Path(short_id): Path<String>,
) -> Result<Json<serde_json::Value>, (StatusCode, Json<serde_json::Value>)> {
    match db::delete_snippet_by_short_id(&state.db, &short_id) {
        Ok(true) => Ok(Json(serde_json::json!({"deleted": true}))),
        Ok(false) => Err((StatusCode::NOT_FOUND, Json(serde_json::json!({"error": "Snippet not found"})))),
        Err(_) => Err((StatusCode::INTERNAL_SERVER_ERROR, Json(serde_json::json!({"error": "Internal server error"})))),
    }
}

async fn api_update_snippet(
    State(state): State<AppState>,
    Path(short_id): Path<String>,
    Json(body): Json<ApiCreateSnippet>,
) -> Result<Json<Snippet>, (StatusCode, Json<serde_json::Value>)> {
    if body.content.len() > state.server_config.max_content_size {
        return Err((
            StatusCode::PAYLOAD_TOO_LARGE,
            Json(serde_json::json!({
                "error": format!("Content too large. Maximum size is {} bytes", state.server_config.max_content_size)
            })),
        ));
    }
    match db::update_snippet_by_short_id(&state.db, &short_id, &body.name, &body.content) {
        Ok(Some(snippet)) => Ok(Json(snippet)),
        Ok(None) => Err((StatusCode::NOT_FOUND, Json(serde_json::json!({"error": "Snippet not found"})))),
        Err(_) => Err((StatusCode::INTERNAL_SERVER_ERROR, Json(serde_json::json!({"error": "Internal server error"})))),
    }
}

fn build_api_routes(state: &AppState) -> Router<AppState> {
    let config = &state.server_config;

    let auth_layer = middleware::from_fn_with_state(state.clone(), require_api_key);

    // /api/snippets — GET (api_list) and POST (api_create)
    let list_authed = config.requires_auth("api_list");
    let create_authed = config.requires_auth("api_create");

    // /api/snippets/{short_id} — GET (api_get), PUT (api_update), and DELETE (api_delete)
    let get_authed = config.requires_auth("api_get");
    let update_authed = config.requires_auth("api_update");
    let delete_authed = config.requires_auth("api_delete");

    // Build authed router
    let mut authed = Router::new();
    if list_authed {
        authed = authed.route("/api/snippets", get(api_list_snippets));
    }
    if create_authed {
        authed = authed.route("/api/snippets", post(api_create_snippet));
    }
    if get_authed {
        authed = authed.route("/api/snippets/{short_id}", get(api_get_snippet));
    }
    if update_authed {
        authed = authed.route("/api/snippets/{short_id}", put(api_update_snippet));
    }
    if delete_authed {
        authed = authed.route("/api/snippets/{short_id}", delete(api_delete_snippet));
    }
    let authed = authed.route_layer(auth_layer);

    // Build open router
    let mut open = Router::new();
    if !list_authed {
        open = open.route("/api/snippets", get(api_list_snippets));
    }
    if !create_authed {
        open = open.route("/api/snippets", post(api_create_snippet));
    }
    if !get_authed {
        open = open.route("/api/snippets/{short_id}", get(api_get_snippet));
    }
    if !update_authed {
        open = open.route("/api/snippets/{short_id}", put(api_update_snippet));
    }
    if !delete_authed {
        open = open.route("/api/snippets/{short_id}", delete(api_delete_snippet));
    }

    authed.merge(open)
}

fn mime_from_path(path: &str) -> &'static str {
    match path.rsplit('.').next().unwrap_or("") {
        "css" => "text/css",
        "js" => "application/javascript",
        "html" => "text/html",
        "png" => "image/png",
        "ico" => "image/x-icon",
        "svg" => "image/svg+xml",
        "woff" => "font/woff",
        "woff2" => "font/woff2",
        "ttf" => "font/ttf",
        "otf" => "font/otf",
        "json" | "webmanifest" => "application/json",
        "jpg" | "jpeg" => "image/jpeg",
        _ => "application/octet-stream",
    }
}

async fn serve_assets(Path(path): Path<String>) -> Response {
    match Assets::get(&path) {
        Some(file) => {
            let mime = mime_from_path(&path);
            ([(header::CONTENT_TYPE, mime)], file.data).into_response()
        }
        None => StatusCode::NOT_FOUND.into_response(),
    }
}

async fn serve_static(Path(path): Path<String>) -> Response {
    match Static::get(&path) {
        Some(file) => {
            let mime = mime_from_path(&path);
            ([(header::CONTENT_TYPE, mime)], file.data).into_response()
        }
        None => StatusCode::NOT_FOUND.into_response(),
    }
}

pub async fn run(host: String, port: u16) {
    dotenvy::dotenv().ok();

    let server_config = ServerConfig::from_env();

    // Validate endpoint names
    let known = ["api_list", "api_create", "api_get", "api_update", "api_delete", "all", "none"];
    for name in &server_config.auth_endpoints {
        if !known.contains(&name.as_str()) {
            eprintln!("Warning: unknown auth endpoint name '{}' in SIPP_AUTH_ENDPOINTS", name);
        }
    }

    if !server_config.auth_endpoints.is_empty() && server_config.api_key.is_none() {
        eprintln!("Warning: SIPP_AUTH_ENDPOINTS is set but SIPP_API_KEY is not configured");
    }

    if server_config.auth_endpoints.is_empty() {
        println!("Auth: disabled (no endpoints require authentication)");
    } else {
        let names: Vec<&str> = server_config.auth_endpoints.iter().map(|s| s.as_str()).collect();
        println!("Auth: enabled for endpoints: {}", names.join(", "));
    }

    println!("Max content size: {} bytes", server_config.max_content_size);

    let state = AppState {
        db: db::init_db().expect("Failed to initialize database"),
        highlighter: Arc::new(Highlighter::new()),
        server_config,
    };

    let api_routes = build_api_routes(&state);

    let app = Router::new()
        .route("/", get(index))
        .route("/admin", get(admin))
        .route("/s/{short_id}", get(view_snippet))
        .route("/snippets", post(create_snippet))
        .merge(api_routes)
        .route("/assets/{*path}", get(serve_assets))
        .route("/static/{*path}", get(serve_static))
        .with_state(state);

    let addr = format!("{}:{}", host, port);
    let listener = tokio::net::TcpListener::bind(&addr)
        .await
        .unwrap_or_else(|_| panic!("Failed to bind to {}", addr));

    println!("Server running at http://{}:{}", host, port);

    axum::serve(listener, app)
        .await
        .expect("Failed to start server");
}
