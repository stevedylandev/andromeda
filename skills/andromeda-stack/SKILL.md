---
name: Andromeda Stack
description: Scaffold a Rust CRUD web application with Axum, SQLite, Askama templates, andromeda-auth session auth, embedded static assets, and Docker deployment. Use when the user wants to build a new Rust web server with CRUD operations in the andromeda monorepo.
---

# Rust Andromeda Web App

## Overview

Scaffold and build Rust CRUD web applications in the andromeda workspace using Axum + SQLite + Askama templates + `andromeda-auth` for authentication. The result is a single binary web server with HTML pages, a JSON API, optional session or API key auth, and Docker deployment.

All apps live under `apps/` in the workspace and share dependencies via the root `Cargo.toml`. Shared crates live under `crates/`:

- `andromeda-auth` — password verify, session token, cookie helpers
- `andromeda-db` — DB connection helpers, optional `session` feature
- `andromeda-darkmatter-css` — shared CSS theme

## Project Structure

```
apps/app-name/
├── src/
│   ├── main.rs         # Entry point, starts the server
│   ├── server.rs       # Axum routes, middleware, handlers, static asset serving
│   ├── db.rs           # SQLite schema, CRUD functions, error types
│   └── auth.rs         # Session/cookie auth wrapper (uses andromeda-auth crate)
├── templates/          # Askama HTML templates
│   ├── base.html       # Base layout with blocks (title, content)
│   └── *.html          # Pages extend base.html
├── static/             # CSS, fonts, favicons (served via tower-http ServeDir)
│   └── styles.css
├── .env.example        # Environment variable reference
├── Dockerfile          # Multi-stage workspace build
└── docker-compose.yml  # Compose config with volume for DB persistence
```

## Dependencies (Cargo.toml)

Apps use workspace dependencies from the root `Cargo.toml`. Use `{ workspace = true }` for shared crates:

```toml
[package]
name = "app-name"
version = "0.1.0"
edition = "2024"
description = "Short app description"
license = "MIT"
repository = "https://github.com/stevedylandev/andromeda"
homepage = "https://github.com/stevedylandev/andromeda"

[dependencies]
axum = { workspace = true }
tokio = { workspace = true }
serde = { workspace = true }
serde_json = { workspace = true }
rusqlite = { workspace = true }
nanoid = { workspace = true }
rust-embed = { workspace = true }
dotenvy = { workspace = true }
subtle = { workspace = true }
rand = { workspace = true }
tracing = { workspace = true }
tracing-subscriber = { workspace = true }
andromeda-auth = { workspace = true }
andromeda-db = { workspace = true, features = ["session"] }  # if using session auth
andromeda-darkmatter-css = { workspace = true }              # if using shared theme
askama = "0.15"
askama_web = { version = "0.15", features = ["axum-0.8"] }
```

After creating the app, register it in the workspace root `Cargo.toml` under `[workspace] members`. See **Wiring up a new app** below for the full set of workspace files to update.

Only add additional crates when the specific app requires them. Do NOT include TUI crates (ratatui, crossterm), CLI crates (clap), or HTTP client crates (reqwest) unless explicitly requested.

- `tracing` + `tracing-subscriber` — structured logging, always include
- `rand` — needed for session token generation when using session-based auth
- `tower-http` — for serving static files from disk via `ServeDir`

## Database Layer (db.rs)

Pattern: single-file module with `Arc<Mutex<Connection>>` for thread-safe SQLite access.

### Structure

```rust
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
```

### Key patterns

- **Model struct**: derive `Serialize, Deserialize`, all fields `pub`
- **ID generation**: `nanoid!(10)` for short unique IDs
- **DB path from env**: `std::env::var("APP_DB_PATH").unwrap_or_else(|_| "app.sqlite".to_string())`
- **init_db()**: opens connection, runs `CREATE TABLE IF NOT EXISTS`, returns `Arc<Mutex<Connection>>`
- **CRUD functions**: standalone functions that take `&Db` as first param
  - `create_*` — INSERT, return created model with `last_insert_rowid()`
  - `get_*_by_short_id` — SELECT, return `Result<Option<Model>, DbError>`
  - `get_all_*` — SELECT with ORDER BY id DESC
  - `delete_*_by_short_id` — DELETE, return `Result<bool, DbError>` (rows_affected > 0)
  - `update_*_by_short_id` — UPDATE then SELECT to return updated model
- **Error handling**: `QueryReturnedNoRows` maps to `Ok(None)`, not an error

## Server Layer (server.rs)

### Embedded assets with rust_embed

```rust
use rust_embed::Embed;

#[derive(Embed)]
#[folder = "assets/"]
struct Assets;

#[derive(Embed)]
#[folder = "static/"]
struct Static;
```

Serve with handlers that match on file path and return correct MIME types:

```rust
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
```

### App state

```rust
#[derive(Clone)]
struct AppState {
    db: Db,
    server_config: ServerConfig,
}
```

Add domain-specific fields as needed (e.g., a highlighter, cache, etc).

### Askama templates

```rust
use askama::Template;
use askama_web::WebTemplate;

#[derive(Template)]
#[template(path = "index.html")]
struct IndexTemplate;

// Templates with data:
#[derive(Template)]
#[template(path = "item.html")]
struct ItemTemplate {
    name: String,
    content: String,
}
```

Render with `WebTemplate(MyTemplate { ... })`.

### Route structure

Two sets of routes: **web routes** (HTML pages + form submissions) and **API routes** (JSON).

**Web routes:**
- `GET /` — index page (template)
- `GET /admin` — admin panel (template)
- `POST /items` — form submission, redirects on success
- `GET /items/{short_id}` — view single item (template)

**API routes:**
- `GET /api/items` — list all (JSON)
- `POST /api/items` — create (JSON body → 201 + JSON)
- `GET /api/items/{short_id}` — get one (JSON)
- `PUT /api/items/{short_id}` — update (JSON body → JSON)
- `DELETE /api/items/{short_id}` — delete (JSON)

**Static asset routes:**
- `GET /assets/{*path}` — embedded assets (favicons, fonts, images)
- `GET /static/{*path}` — embedded static files (CSS)

### Form deserialization

```rust
#[derive(Deserialize)]
struct CreateItemForm {
    name: String,
    content: String,
}
```

Use `Form(form): Form<CreateItemForm>` for HTML forms, `Json(body): Json<CreateItem>` for API.

### Error responses

- Web handlers return `Result<..., (StatusCode, Html<String>)>`
- API handlers return `Result<..., (StatusCode, Json<serde_json::Value>)>`
- Use `serde_json::json!({"error": "message"})` for API error bodies

## Authentication

The workspace provides a shared `andromeda-auth` crate at `crates/auth/` with these primitives:

```rust
// andromeda-auth public API
pub fn verify_password(input: &str, expected: &str) -> bool;
pub fn generate_session_token() -> String;
pub fn build_session_cookie(token: &str, secure: bool) -> String;
pub fn clear_session_cookie() -> String;
pub fn extract_session_cookie(headers: &axum::http::HeaderMap) -> Option<String>;
```

Apps import `andromeda-auth` via workspace dependency and wrap it in a local `auth.rs` module.

### Session/cookie auth (for web-facing apps)

The standard pattern for apps that need login (e.g., jotts, parcels, feeds). Create a `src/auth.rs` that wraps the auth crate with app-specific session storage and an Axum extractor.

**Sessions table in db.rs:**

```sql
CREATE TABLE IF NOT EXISTS sessions (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    token      TEXT NOT NULL UNIQUE,
    expires_at TEXT NOT NULL
);
```

**Session DB functions in db.rs:**

```rust
pub fn insert_session(db: &Db, token: &str, expires_at: &str) -> Result<(), DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    conn.execute("INSERT INTO sessions (token, expires_at) VALUES (?1, ?2)", params![token, expires_at])?;
    Ok(())
}

pub fn get_session_expiry(db: &Db, token: &str) -> Result<Option<String>, DbError> {
    let conn = db.lock().map_err(|_| DbError::LockPoisoned)?;
    match conn.query_row("SELECT expires_at FROM sessions WHERE token = ?1", params![token], |row| row.get(0)) {
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
    conn.execute("DELETE FROM sessions WHERE expires_at < datetime('now')", [])?;
    Ok(())
}
```

**auth.rs module — wraps andromeda-auth with session validation:**

```rust
use axum::{
    extract::{FromRef, FromRequestParts},
    http::request::Parts,
    response::{IntoResponse, Redirect, Response},
};
use std::sync::Arc;

use crate::AppState;

/// Axum extractor — guards routes behind login. Redirects to /login if invalid.
pub struct AuthSession;

impl<S> FromRequestParts<S> for AuthSession
where
    S: Send + Sync,
    Arc<AppState>: FromRef<S>,
{
    type Rejection = Response;

    async fn from_request_parts(parts: &mut Parts, state: &S) -> Result<Self, Self::Rejection> {
        let state = Arc::<AppState>::from_ref(state);
        let token = andromeda_auth::extract_session_cookie(&parts.headers);
        if let Some(token) = token {
            if is_valid_session(&state, &token) {
                return Ok(AuthSession);
            }
        }
        Err(Redirect::to("/login").into_response())
    }
}

fn is_valid_session(state: &AppState, token: &str) -> bool {
    // Check DB for token expiry — return true if token exists and hasn't expired
    match crate::db::get_session_expiry(&state.db, token) {
        Ok(Some(expires_at)) => expires_at > chrono::Utc::now().to_rfc3339(),
        _ => false,
    }
}
```

**Usage in handlers** — use `andromeda_auth` functions directly for login/logout:

```rust
use andromeda_auth;

// POST /login — verify password, create session, set cookie
async fn post_login(State(state): State<Arc<AppState>>, Form(form): Form<LoginForm>) -> Response {
    if andromeda_auth::verify_password(&form.password, &state.app_password) {
        let token = andromeda_auth::generate_session_token();
        // Store token in DB with expiry...
        let cookie = andromeda_auth::build_session_cookie(&token, state.cookie_secure);
        // Set cookie header and redirect to /
    } else {
        // Redirect to /login?error=Invalid+password
    }
}

// GET /logout — clear session
async fn get_logout(headers: HeaderMap, State(state): State<Arc<AppState>>) -> Response {
    if let Some(token) = andromeda_auth::extract_session_cookie(&headers) {
        let _ = crate::db::delete_session(&state.db, &token);
    }
    let cookie = andromeda_auth::clear_session_cookie();
    // Set cookie header and redirect to /login
}
```

**Protect routes** — add `_session: auth::AuthSession` as a parameter:

```rust
async fn get_index(_session: auth::AuthSession, State(state): State<Arc<AppState>>) -> Response {
    // only reachable if session is valid
}
```

**AppState for session auth:**

```rust
pub struct AppState {
    pub db: Db,
    pub app_password: String,
    pub cookie_secure: bool,
}
```

### API key auth (alternative — for API-only apps)

For apps that don't need a login page (e.g., sipp), use API key middleware instead of session auth. This pattern does NOT use `andromeda-auth` — it's self-contained in `server.rs`:

```rust
#[derive(Clone)]
struct ServerConfig {
    api_key: Option<String>,
    auth_endpoints: HashSet<String>,
}
```

See `apps/sipp` for the full API key middleware pattern.

### Environment variables

Prefix app-specific env vars with the app name (e.g., `JOTTS_`, `SIPP_`). Shared vars like `HOST`, `PORT`, `COOKIE_SECURE` don't need a prefix:

| Variable | Purpose | Default |
|----------|---------|---------|
| `HOST` | Bind address | `127.0.0.1` (set to `0.0.0.0` in Docker) |
| `PORT` | Listen port | `3000` |
| `APP_DB_PATH` | SQLite file path | `app.sqlite` |
| `APP_PASSWORD` | Single password for session auth (web apps) | None |
| `COOKIE_SECURE` | Set `true` for HTTPS-only cookies | `false` |
| `APP_API_KEY` | API key for API-key auth pattern | None (auth disabled) |
| `APP_AUTH_ENDPOINTS` | Comma-separated endpoint names, "all", or "none" | `api_delete,api_list,api_update` |

## Templates (Askama)

HTML templates live in `templates/` and use Askama syntax. Key patterns:

- Link CSS via `/static/styles.css`
- Link assets via `/assets/filename`
- Include `<meta name="theme-color" content="#121113" />`
- Forms POST to web routes (not API routes)
- Use `{{ variable }}` for template interpolation
- Use `{{ variable|safe }}` for pre-rendered HTML (e.g., syntax highlighted content)

### Template inheritance

Use a `base.html` with block sections. All pages extend it:

**templates/base.html:**
```html
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>{% block title %}APP_NAME{% endblock %}</title>
  <meta name="theme-color" content="#121113" />
  <style>
    /* base styles here */
  </style>
</head>
<body>
  <div class="container">
    {% block content %}{% endblock %}
  </div>
</body>
</html>
```

**templates/index.html:**
```html
{% extends "base.html" %}
{% block title %}Items{% endblock %}
{% block content %}
  {% if let Some(error) = error %}
    <p class="error">{{ error }}</p>
  {% endif %}
  {% for item in items %}
    <div>{{ item.name }}</div>
  {% endfor %}
{% endblock %}
```

### Flash messages via query params

Pass transient error/success messages through redirects using query parameters. No session flash needed.

**Query param struct:**
```rust
#[derive(Deserialize, Default)]
pub struct FlashQuery {
    pub error: Option<String>,
}
```

**In handlers** — redirect with message:
```rust
Redirect::to("/items/add?error=Name+is+required.").into_response()
```

**In receiving handler** — extract and pass to template:
```rust
async fn get_add(Query(q): Query<FlashQuery>) -> Response {
    render(AddTemplate { error: q.error })
}
```

**In template** — conditionally render:
```html
{% if let Some(error) = error %}
  <p class="error">{{ error }}</p>
{% endif %}
```

## Logging (tracing)

Always initialize tracing in `main()` before anything else:

```rust
tracing_subscriber::fmt::init();
```

Use throughout the app:
- `tracing::error!("DB error: {}", e)` — unrecoverable failures
- `tracing::warn!("Non-critical issue: {}", e)` — degraded but functional
- `tracing::info!("Listening on {}", addr)` — startup/lifecycle events

## main.rs

Minimal — just loads env and starts the server:

```rust
mod auth;
mod db;
mod server;

#[tokio::main]
async fn main() {
    dotenvy::dotenv().ok();
    tracing_subscriber::fmt::init();
    let host = std::env::var("HOST").unwrap_or_else(|_| "127.0.0.1".to_string());
    let port: u16 = std::env::var("PORT")
        .ok()
        .and_then(|v| v.parse().ok())
        .unwrap_or(3000);
    server::run(host, port).await;
}
```

Keep main.rs minimal — all logic lives in `server.rs`, `db.rs`, and `auth.rs`.

## Dockerfile

Multi-stage workspace build using `cargo-chef` for dep caching. Built from repo root with `docker build -f apps/APP_NAME/Dockerfile .`:

```dockerfile
# Build from repo root: docker build -t APP_NAME -f apps/APP_NAME/Dockerfile .
FROM lukemathwalker/cargo-chef:latest-rust-1-slim-bookworm AS chef
WORKDIR /app

FROM chef AS planner
COPY . .
RUN cargo chef prepare --recipe-path recipe.json

FROM chef AS builder
RUN apt-get update && apt-get install -y pkg-config libssl-dev && rm -rf /var/lib/apt/lists/*
COPY --from=planner /app/recipe.json recipe.json
RUN cargo chef cook --release --recipe-path recipe.json -p APP_NAME
COPY . .
RUN cargo build --release -p APP_NAME

FROM debian:bookworm-slim
COPY --from=builder /app/target/release/APP_NAME /usr/local/bin/APP_NAME
WORKDIR /data
EXPOSE 3000
ENV HOST=0.0.0.0
ENV PORT=3000
CMD ["APP_NAME"]
```

Replace all `APP_NAME` with actual binary/package name. If app makes HTTPS requests (uses reqwest etc.), add `RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*` to final stage.

## docker-compose.yml

```yaml
services:
  app:
    build:
      context: ../..
      dockerfile: apps/APP_NAME/Dockerfile
    ports:
      - "${PORT:-3000}:${PORT:-3000}"
    environment:
      - APP_PASSWORD=${APP_PASSWORD:-changeme}
      - APP_DB_PATH=/data/APP_NAME.sqlite
      - COOKIE_SECURE=false
      - HOST=0.0.0.0
      - PORT=${PORT:-3000}
    volumes:
      - app-data:/data
    restart: unless-stopped

volumes:
  app-data:
```

Key points:
- `context: ../..` builds from the workspace root
- Named volume persists the SQLite database across container restarts
- ENV vars use app-specific prefixes (e.g., `JOTTS_PASSWORD`, `SIPP_API_KEY`)

## .env.example

Always create one with all configurable env vars and sensible comments.

## Wiring up a new app

A new app is not done when `cargo run -p APP_NAME` works. Several workspace-level files list every app explicitly and must be updated, or CI/release/deploy break silently.

### 1. Workspace root `Cargo.toml`

Add to `[workspace] members`:

```toml
[workspace]
members = [
    "apps/sipp",
    # ...
    "apps/APP_NAME",
    "crates/auth",
    "crates/db",
    "crates/darkmatter-css",
]
```

### 2. `dist-workspace.toml`

Add to `[workspace] members` so `cargo dist` builds release binaries + Homebrew formula:

```toml
[workspace]
members = [
    "cargo:apps/sipp",
    # ...
    "cargo:apps/APP_NAME",
]
```

### 3. `.github/workflows/docker.yml`

Two spots — both list every app:

```yaml
ALL='["cellar","sipp","feeds","parcels","jotts","og","shrink","backup","posts","library","APP_NAME"]'
```

```yaml
for app in cellar sipp feeds parcels jotts og shrink backup posts library APP_NAME; do
```

If the cargo package name differs from directory name, add a case to `pkg_to_dir()`:

```bash
pkg_to_dir() {
  case "$1" in
    sipp-so) echo "sipp" ;;
    APP_PKG) echo "APP_NAME" ;;
    *) echo "$1" ;;
  esac
}
```

### 4. `.github/workflows/docker-test.yml`

Same two lists — keep in sync with `docker.yml`:

```yaml
ALL='["cellar","sipp","feeds","parcels","jotts","og","shrink","backup","posts","library","APP_NAME"]'
```

```yaml
for app in cellar sipp feeds parcels jotts og shrink backup posts library APP_NAME; do
```

### 5. Root `docker-compose.yml`

Production compose at repo root pulls images from GHCR. Add a service + volume:

```yaml
services:
  APP_NAME:
    image: ghcr.io/stevedylandev/andromeda/APP_NAME:latest
    restart: unless-stopped
    ports:
      - "PORT:3000"
    volumes:
      - APP_NAME_data:/data
    env_file: apps/APP_NAME/.env

volumes:
  APP_NAME_data:
    external: true
    name: APP_NAME_APP_NAME-data
```

Pick a unique host port (existing: 3000 sipp, 3030 jotts, 3456 og, 3535 posts, 3565 shrink, 4555 feeds, 4646 library, 6754 cellar). Drop `volumes`/`env_file` if app is stateless (see `shrink`, `og`).

If the app holds data worth backing up, add a read-only mount to the `backup` service:

```yaml
backup:
  volumes:
    - APP_NAME_data:/data/APP_NAME:ro
```

### 6. App-local `docker-compose.yml`

Per-app compose for local dev — builds from source:

```yaml
services:
  app:
    build:
      context: ../..
      dockerfile: apps/APP_NAME/Dockerfile
    ports:
      - "${PORT:-3000}:${PORT:-3000}"
    environment:
      - APP_PASSWORD=${APP_PASSWORD:-changeme}
      - APP_DB_PATH=/data/APP_NAME.sqlite
      - COOKIE_SECURE=false
      - HOST=0.0.0.0
      - PORT=${PORT:-3000}
    volumes:
      - APP_NAME-data:/data
    restart: unless-stopped

volumes:
  APP_NAME-data:
```

## Checklist

When scaffolding a new app:

1. Create `apps/APP_NAME/` with `cargo init`
2. Set up `apps/APP_NAME/Cargo.toml` with workspace deps + `andromeda-auth` (and `andromeda-db`, `andromeda-darkmatter-css` if used)
3. Register app in workspace root `Cargo.toml` under `[workspace] members`
4. Register app in `dist-workspace.toml` under `[workspace] members` (with `cargo:` prefix)
5. Add app to `ALL` array + `for app in ...` loop in `.github/workflows/docker.yml`
6. Add app to `ALL` array + `for app in ...` loop in `.github/workflows/docker-test.yml`
7. Write `db.rs` — schema, model, CRUD, session table if needed
8. Write `auth.rs` — wrap `andromeda-auth` with `AuthSession` extractor (if auth needed)
9. Write `server.rs` — config, state, templates, handlers, routes
10. Write `main.rs` — minimal entry point
11. Create `templates/` with `base.html` + index page
12. Create `static/styles.css`
13. Create `.env.example`
14. Create `Dockerfile` (cargo-chef multi-stage) + app-local `docker-compose.yml`
15. Add service entry + volume to root `docker-compose.yml` (and `backup` mount if stateful)
16. Test: `cargo run -p APP_NAME`, hit routes
17. Test: `docker build -f apps/APP_NAME/Dockerfile .` from repo root

## What NOT to include

- No external CSS frameworks unless specified 
- No ORMs — use raw rusqlite
- No connection pools — `Arc<Mutex<Connection>>` is sufficient for SQLite
- No async database drivers — rusqlite is synchronous and that's fine
