mod db;
mod auth;
mod usps;

use askama::Template;
use axum::{
    Form,
    Router,
    extract::{Path, Query, State},
    http::{HeaderMap, StatusCode},
    response::{Html, IntoResponse, Redirect, Response},
    routing::{get, post},
};
use db::Db;
use reqwest::Client;
use serde::Deserialize;
use std::sync::{Arc, Mutex};
use tower_http::services::ServeDir;

// ── App State ──────────────────────────────────────────────────────────────

pub struct AppState {
    pub db: Db,
    pub app_password: String,
    pub cookie_secure: bool,
    pub usps_token: Arc<Mutex<Option<usps::CachedToken>>>,
    pub usps_client_id: String,
    pub usps_client_secret: String,
    pub http_client: Client,
}

// ── Template rendering helper ──────────────────────────────────────────────

fn render<T: Template>(t: T) -> Response {
    match t.render() {
        Ok(html) => Html(html).into_response(),
        Err(e) => {
            tracing::error!("Template render error: {}", e);
            (StatusCode::INTERNAL_SERVER_ERROR, "Internal error.").into_response()
        }
    }
}

// ── Query params ───────────────────────────────────────────────────────────

#[derive(Deserialize, Default)]
pub struct ErrorQuery {
    pub error: Option<String>,
}

// ── Templates ──────────────────────────────────────────────────────────────

#[derive(Template)]
#[template(path = "login.html")]
struct LoginTemplate {
    error: Option<String>,
}

#[derive(Template)]
#[template(path = "index.html")]
struct IndexTemplate {
    packages: Vec<db::Package>,
    error: Option<String>,
}

#[derive(Template)]
#[template(path = "add.html")]
struct AddTemplate {
    error: Option<String>,
}

#[derive(Template)]
#[template(path = "detail.html")]
struct DetailTemplate {
    package: db::Package,
    events: Vec<db::TrackingEvent>,
    error: Option<String>,
}

// ── Login ──────────────────────────────────────────────────────────────────

async fn get_login(Query(q): Query<ErrorQuery>) -> Response {
    render(LoginTemplate { error: q.error })
}

#[derive(Deserialize)]
struct LoginForm {
    password: String,
}

async fn post_login(
    State(state): State<Arc<AppState>>,
    Form(form): Form<LoginForm>,
) -> Response {
    if !auth::verify_password(&form.password, &state.app_password) {
        return render(LoginTemplate { error: Some("Invalid password.".to_string()) });
    }

    let _ = db::prune_expired_sessions(&state.db);

    let token = auth::generate_session_token();
    let expires_at = auth::session_expiry_at();

    if let Err(e) = db::insert_session(&state.db, &token, &expires_at) {
        tracing::error!("Failed to insert session: {}", e);
        return (StatusCode::INTERNAL_SERVER_ERROR, "Internal error.").into_response();
    }

    let cookie = auth::build_session_cookie(&token, state.cookie_secure);
    let mut resp = Redirect::to("/").into_response();
    resp.headers_mut().insert("set-cookie", cookie.parse().unwrap());
    resp
}

// ── Logout ─────────────────────────────────────────────────────────────────

async fn get_logout(
    State(state): State<Arc<AppState>>,
    headers: HeaderMap,
) -> Response {
    if let Some(token) = auth::extract_session_token(&headers) {
        let _ = db::delete_session(&state.db, &token);
    }
    let mut resp = Redirect::to("/login").into_response();
    resp.headers_mut().insert("set-cookie", auth::clear_session_cookie().parse().unwrap());
    resp
}

// ── Refresh Helper ─────────────────────────────────────────────────────────

async fn refresh_one(state: &AppState, package: &db::Package) -> Result<(), anyhow::Error> {
    let token = usps::get_token(
        &state.usps_token,
        &state.http_client,
        &state.usps_client_id,
        &state.usps_client_secret,
    )
    .await?;

    let detail = usps::fetch_tracking(&state.http_client, &token, &package.tracking_number).await?;

    use std::time::{SystemTime, UNIX_EPOCH};
    let now = SystemTime::now().duration_since(UNIX_EPOCH).unwrap().as_secs();
    let refreshed_at = {
        let days = now / 86400;
        let h = (now / 3600) % 24;
        let m = (now / 60) % 60;
        let s = now % 60;
        auth::format_unix_to_datetime_pub(days, h, m, s)
    };

    let expected_delivery = detail
        .delivery_date_expectation
        .as_ref()
        .and_then(|d| d.expected_delivery_date.as_deref());

    db::update_package_status(
        &state.db,
        package.id,
        detail.status.as_deref().unwrap_or(""),
        detail.status_category.as_deref(),
        detail.status_summary.as_deref(),
        detail.mail_class.as_deref(),
        expected_delivery,
        &refreshed_at,
    )?;

    db::delete_events_for_package(&state.db, package.id)?;

    for event in &detail.tracking_events {
        if let Err(e) = db::insert_event(
            &state.db,
            package.id,
            event.event_timestamp.as_deref(),
            event.event_type.as_deref(),
            event.event_city.as_deref(),
            event.event_state.as_deref(),
            event.event_zip_code.as_deref(),
            event.event_code.as_deref(),
        ) {
            tracing::warn!("DB error inserting event for package {}: {}", package.id, e);
        }
    }

    Ok(())
}

// ── Index ──────────────────────────────────────────────────────────────────

async fn get_index(
    _session: auth::AuthSession,
    State(state): State<Arc<AppState>>,
    Query(q): Query<ErrorQuery>,
) -> Response {
    let packages = match db::list_packages(&state.db) {
        Ok(p) => p,
        Err(e) => {
            tracing::error!("DB error listing packages: {}", e);
            return (StatusCode::INTERNAL_SERVER_ERROR, "Internal error.").into_response();
        }
    };

    for package in &packages {
        if let Err(e) = refresh_one(&state, package).await {
            tracing::warn!("Failed to refresh package {}: {}", package.id, e);
        }
    }

    match db::list_packages(&state.db) {
        Ok(packages) => render(IndexTemplate { packages, error: q.error }),
        Err(e) => {
            tracing::error!("DB error listing packages after refresh: {}", e);
            (StatusCode::INTERNAL_SERVER_ERROR, "Internal error.").into_response()
        }
    }
}

// ── Add Package ────────────────────────────────────────────────────────────

async fn get_add(
    _session: auth::AuthSession,
    Query(q): Query<ErrorQuery>,
) -> Response {
    render(AddTemplate { error: q.error })
}

#[derive(Deserialize)]
struct AddPackageForm {
    tracking_number: String,
    label: Option<String>,
}

async fn post_packages(
    _session: auth::AuthSession,
    State(state): State<Arc<AppState>>,
    Form(form): Form<AddPackageForm>,
) -> Response {
    let tracking_number = form.tracking_number.trim().to_uppercase();
    if tracking_number.is_empty() {
        return Redirect::to("/packages/add?error=Tracking+number+is+required.").into_response();
    }
    let label = form.label.as_deref().map(str::trim).filter(|s| !s.is_empty());

    match db::insert_package(&state.db, &tracking_number, label) {
        Ok(_) => Redirect::to("/").into_response(),
        Err(e) if e.to_string().contains("UNIQUE") => {
            Redirect::to("/packages/add?error=Tracking+number+already+exists.").into_response()
        }
        Err(e) => {
            tracing::error!("DB error inserting package: {}", e);
            (StatusCode::INTERNAL_SERVER_ERROR, "Internal error.").into_response()
        }
    }
}

// ── Delete Package ─────────────────────────────────────────────────────────

async fn post_delete_package(
    _session: auth::AuthSession,
    State(state): State<Arc<AppState>>,
    Path(id): Path<i64>,
) -> Response {
    if let Err(e) = db::delete_package(&state.db, id) {
        tracing::error!("DB error deleting package {}: {}", id, e);
        return (StatusCode::INTERNAL_SERVER_ERROR, "Internal error.").into_response();
    }
    Redirect::to("/").into_response()
}

// ── Refresh Package ────────────────────────────────────────────────────────

async fn post_refresh_package(
    _session: auth::AuthSession,
    State(state): State<Arc<AppState>>,
    Path(id): Path<i64>,
) -> Response {
    let package = match db::get_package(&state.db, id) {
        Ok(Some(p)) => p,
        Ok(None) => return Redirect::to("/?error=Package+not+found.").into_response(),
        Err(e) => {
            tracing::error!("DB error fetching package {}: {}", id, e);
            return (StatusCode::INTERNAL_SERVER_ERROR, "Internal error.").into_response();
        }
    };

    if let Err(e) = refresh_one(&state, &package).await {
        let msg = urlencoding_encode(&e.to_string());
        return Redirect::to(&format!("/packages/{}?error={}", id, msg)).into_response();
    }

    Redirect::to(&format!("/packages/{}", id)).into_response()
}

// ── Package Detail ─────────────────────────────────────────────────────────

async fn get_package_detail(
    _session: auth::AuthSession,
    State(state): State<Arc<AppState>>,
    Path(id): Path<i64>,
    Query(q): Query<ErrorQuery>,
) -> Response {
    let package = match db::get_package(&state.db, id) {
        Ok(Some(p)) => p,
        Ok(None) => return Redirect::to("/?error=Package+not+found.").into_response(),
        Err(e) => {
            tracing::error!("DB error fetching package {}: {}", id, e);
            return (StatusCode::INTERNAL_SERVER_ERROR, "Internal error.").into_response();
        }
    };

    let events = match db::get_events_for_package(&state.db, id) {
        Ok(e) => e,
        Err(e) => {
            tracing::error!("DB error fetching events for package {}: {}", id, e);
            vec![]
        }
    };

    render(DetailTemplate { package, events, error: q.error })
}

// ── URL encoding helper ────────────────────────────────────────────────────

fn urlencoding_encode(s: &str) -> String {
    s.chars()
        .flat_map(|c| match c {
            ' ' => vec!['+'],
            c if c.is_ascii_alphanumeric() || c == '-' || c == '_' || c == '.' || c == '~' => vec![c],
            c => {
                let mut buf = [0u8; 4];
                let bytes = c.encode_utf8(&mut buf);
                bytes.bytes().flat_map(|b| {
                    vec!['%', char::from_digit((b >> 4) as u32, 16).unwrap().to_ascii_uppercase(),
                         char::from_digit((b & 0xf) as u32, 16).unwrap().to_ascii_uppercase()]
                }).collect()
            }
        })
        .collect()
}

// ── Main ───────────────────────────────────────────────────────────────────

#[tokio::main]
async fn main() {
    tracing_subscriber::fmt::init();
    use std::env;

    let database_url = db::database_path();
    let app_password = env::var("APP_PASSWORD").expect("APP_PASSWORD must be set");
    let usps_client_id = env::var("USPS_CLIENT_ID").expect("USPS_CLIENT_ID must be set");
    let usps_client_secret = env::var("USPS_CLIENT_SECRET").expect("USPS_CLIENT_SECRET must be set");
    let bind_addr = env::var("BIND_ADDR").unwrap_or_else(|_| "0.0.0.0:3012".to_string());
    let cookie_secure = env::var("COOKIE_SECURE")
        .map(|v| v.eq_ignore_ascii_case("true"))
        .unwrap_or(false);

    let db = db::init_db(&database_url).expect("Failed to open database");

    let state = Arc::new(AppState {
        db,
        app_password,
        cookie_secure,
        usps_token: Arc::new(Mutex::new(None)),
        usps_client_id,
        usps_client_secret,
        http_client: Client::new(),
    });

    let app = Router::new()
        .route("/login", get(get_login).post(post_login))
        .route("/logout", get(get_logout))
        .route("/", get(get_index))
        .route("/packages/add", get(get_add))
        .route("/packages", post(post_packages))
        .route("/packages/{id}", get(get_package_detail))
        .route("/packages/{id}/refresh", post(post_refresh_package))
        .route("/packages/{id}/delete", post(post_delete_package))
        .nest_service("/static", ServeDir::new("static"))
        .with_state(state);

    let listener = tokio::net::TcpListener::bind(&bind_addr)
        .await
        .expect("Failed to bind");
    eprintln!("Listening on {}", bind_addr);
    axum::serve(listener, app).await.expect("Server failed");
}
