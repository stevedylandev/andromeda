mod api;
mod auth;
mod feeds;
mod models;
mod poller;

use std::collections::HashMap;
use std::sync::{Arc, Mutex};

use andromeda_db::{
    feeds as fdb,
    session::{SESSION_SCHEMA, prune_expired_sessions},
    Db,
};
use askama::Template;
use axum::{
    extract::{Multipart, Path, Query, State},
    http::{header, HeaderMap, StatusCode},
    response::{Html, IntoResponse, Json, Redirect, Response},
    routing::{delete, get, post},
    Form, Router,
};
use chrono::DateTime;
use rust_embed::Embed;
use rusqlite::Connection;
use serde::Deserialize;

use crate::poller::POLL_INTERVAL_KEY;

#[derive(Embed)]
#[folder = "static/"]
struct Static;

pub struct AppState {
    pub db: Db,
    pub admin_password: Option<String>,
    pub api_key: Option<String>,
    pub cookie_secure: bool,
    pub base_url: String,
    pub default_poll_minutes: u64,
    pub item_cap: usize,
}

struct TemplateFeedItem {
    title: String,
    link: String,
    author: String,
    formatted_date: String,
}

#[derive(Template)]
#[template(path = "index.html")]
struct IndexTemplate {
    base_url: String,
    items: Vec<TemplateFeedItem>,
    feed_urls: Option<Vec<String>>,
    error: Option<String>,
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
    subscriptions: Vec<AdminSubRow>,
    categories: Vec<fdb::Category>,
    poll_interval_minutes: u64,
    item_cap: usize,
    api_key_configured: bool,
}

struct AdminSubRow {
    id: i64,
    title: String,
    feed_url: String,
    category_name: Option<String>,
    last_fetched_at: Option<String>,
    last_error: Option<String>,
}

fn format_date(timestamp: i64) -> String {
    DateTime::from_timestamp(timestamp, 0)
        .map(|dt| dt.format("%b %-d, %Y").to_string())
        .unwrap_or_default()
}

// ── Public pages ──────────────────────────────────────────────────────

async fn index_handler(
    State(state): State<Arc<AppState>>,
    Query(params): Query<HashMap<String, String>>,
) -> Response {
    let url_query = params.get("url").or_else(|| params.get("urls"));

    let (items, feed_urls, error) = if let Some(query) = url_query {
        let urls: Vec<String> = query
            .split(',')
            .map(|u| u.trim().to_string())
            .filter(|u| !u.is_empty())
            .collect();
        if urls.is_empty() {
            (Vec::new(), None, Some("No URLs provided".to_string()))
        } else {
            let items = feeds::preview_urls(&urls)
                .await
                .into_iter()
                .map(|item| TemplateFeedItem {
                    title: item.title,
                    link: item.link,
                    author: item.author,
                    formatted_date: format_date(item.published),
                })
                .collect();
            (items, Some(urls), None)
        }
    } else {
        match fdb::list_items(
            &state.db,
            &fdb::ListItemsFilter {
                limit: Some(100),
                ..Default::default()
            },
        ) {
            Ok(items) => {
                let rows = items
                    .into_iter()
                    .map(|i| TemplateFeedItem {
                        title: i.title,
                        link: i.link,
                        author: match i.author {
                            Some(a) if !a.is_empty() => format!("{} - {}", i.feed_title, a),
                            _ => i.feed_title,
                        },
                        formatted_date: format_date(i.published_at),
                    })
                    .collect();
                (rows, None, None)
            }
            Err(e) => {
                tracing::error!("index query failed: {e}");
                (
                    Vec::new(),
                    None,
                    Some("Error loading feeds. Please try again later.".to_string()),
                )
            }
        }
    };

    Html(
        IndexTemplate {
            base_url: state.base_url.clone(),
            items,
            feed_urls,
            error,
        }
        .render()
        .unwrap(),
    )
    .into_response()
}

/// Export current subscriptions as OPML.
async fn feeds_opml_handler(State(state): State<Arc<AppState>>) -> Response {
    let subs = match fdb::list_subscriptions(&state.db) {
        Ok(s) => s,
        Err(e) => {
            tracing::error!("opml export failed: {e}");
            return StatusCode::INTERNAL_SERVER_ERROR.into_response();
        }
    };
    let cats: HashMap<i64, String> = fdb::list_categories(&state.db)
        .unwrap_or_default()
        .into_iter()
        .map(|c| (c.id, c.name))
        .collect();

    let now = chrono::Utc::now().to_rfc2822();
    let mut by_cat: HashMap<String, Vec<&fdb::Subscription>> = HashMap::new();
    for sub in &subs {
        let key = sub
            .category_id
            .and_then(|id| cats.get(&id).cloned())
            .unwrap_or_default();
        by_cat.entry(key).or_default().push(sub);
    }

    let mut opml = format!(
        "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<opml version=\"2.0\">\n  <head>\n    <title>Feeds</title>\n    <dateCreated>{now}</dateCreated>\n  </head>\n  <body>\n"
    );

    let mut keys: Vec<&String> = by_cat.keys().collect();
    keys.sort();
    for key in keys {
        let subs = &by_cat[key];
        let indent = if key.is_empty() { "    " } else { "      " };
        if !key.is_empty() {
            opml.push_str(&format!(
                "    <outline text=\"{}\" title=\"{}\">\n",
                escape_xml(key),
                escape_xml(key)
            ));
        }
        for sub in subs {
            opml.push_str(&format!(
                "{indent}<outline type=\"rss\" text=\"{}\" title=\"{}\" xmlUrl=\"{}\" htmlUrl=\"{}\" />\n",
                escape_xml(&sub.title),
                escape_xml(&sub.title),
                escape_xml(&sub.feed_url),
                escape_xml(sub.site_url.as_deref().unwrap_or("")),
            ));
        }
        if !key.is_empty() {
            opml.push_str("    </outline>\n");
        }
    }

    opml.push_str("  </body>\n</opml>");

    (
        [
            (header::CONTENT_TYPE, "application/xml"),
            (
                header::CONTENT_DISPOSITION,
                "attachment; filename=\"feeds.opml\"",
            ),
        ],
        opml,
    )
        .into_response()
}

fn escape_xml(s: &str) -> String {
    s.replace('&', "&amp;")
        .replace('<', "&lt;")
        .replace('>', "&gt;")
        .replace('"', "&quot;")
        .replace('\'', "&apos;")
}

async fn static_handler(axum::extract::Path(path): axum::extract::Path<String>) -> Response {
    match Static::get(&path) {
        Some(file) => {
            let mime = mime_guess::from_path(&path).first_or_octet_stream();
            ([(header::CONTENT_TYPE, mime.as_ref())], file.data.to_vec()).into_response()
        }
        None => StatusCode::NOT_FOUND.into_response(),
    }
}

// ── Admin UI ──────────────────────────────────────────────────────────

#[derive(Deserialize, Default)]
struct FlashQuery {
    error: Option<String>,
    success: Option<String>,
}

#[derive(Deserialize)]
struct LoginForm {
    password: String,
}

#[derive(Deserialize)]
struct AddFeedForm {
    feed_url: String,
    category_name: Option<String>,
}

#[derive(Deserialize)]
struct DiscoverFeedsForm {
    base_url: String,
}

#[derive(Deserialize)]
struct AddCategoryForm {
    name: String,
}

#[derive(Deserialize)]
struct UpdateSubCategoryForm {
    category_name: Option<String>,
}

#[derive(Deserialize)]
struct UpdateSettingsForm {
    poll_interval_minutes: u64,
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

async fn admin_handler(
    _session: auth::AuthSession,
    State(state): State<Arc<AppState>>,
    Query(q): Query<FlashQuery>,
) -> Response {
    let subs = fdb::list_subscriptions(&state.db).unwrap_or_default();
    let cats = fdb::list_categories(&state.db).unwrap_or_default();
    let cat_map: HashMap<i64, String> =
        cats.iter().map(|c| (c.id, c.name.clone())).collect();

    let subscriptions = subs
        .into_iter()
        .map(|s| AdminSubRow {
            id: s.id,
            title: s.title,
            feed_url: s.feed_url,
            category_name: s.category_id.and_then(|id| cat_map.get(&id).cloned()),
            last_fetched_at: s.last_fetched_at,
            last_error: s.last_error,
        })
        .collect();

    let poll_interval_minutes = fdb::get_setting(&state.db, POLL_INTERVAL_KEY)
        .ok()
        .flatten()
        .and_then(|v| v.parse::<u64>().ok())
        .unwrap_or(state.default_poll_minutes);

    Html(
        AdminTemplate {
            success: q.success,
            error: q.error,
            subscriptions,
            categories: cats,
            poll_interval_minutes,
            item_cap: state.item_cap,
            api_key_configured: state.api_key.is_some(),
        }
        .render()
        .unwrap(),
    )
    .into_response()
}

async fn discover_feeds_handler(
    _session: auth::AuthSession,
    Form(form): Form<DiscoverFeedsForm>,
) -> Response {
    match feeds::discover_feeds(&form.base_url).await {
        Ok(urls) => Json(serde_json::json!(urls)).into_response(),
        Err(e) => (
            StatusCode::BAD_REQUEST,
            Json(serde_json::json!({ "error": e })),
        )
            .into_response(),
    }
}

async fn add_feed_handler(
    _session: auth::AuthSession,
    State(state): State<Arc<AppState>>,
    Form(form): Form<AddFeedForm>,
) -> Response {
    let body = api::CreateSubscriptionBody {
        feed_url: form.feed_url,
        title: None,
        category_id: None,
        category_name: form.category_name.filter(|s| !s.trim().is_empty()),
    };
    let resp = api::add_subscription(&state, &body).await;
    let status = resp.status();
    if status.is_success() {
        Redirect::to("/admin?success=Feed+added").into_response()
    } else if status == StatusCode::CONFLICT {
        Redirect::to("/admin?error=Already+subscribed").into_response()
    } else {
        Redirect::to("/admin?error=Failed+to+add+feed").into_response()
    }
}

async fn delete_feed_handler(
    _session: auth::AuthSession,
    State(state): State<Arc<AppState>>,
    Path(id): Path<i64>,
) -> Response {
    match fdb::delete_subscription(&state.db, id) {
        Ok(true) => Redirect::to("/admin?success=Feed+removed").into_response(),
        _ => Redirect::to("/admin?error=Failed+to+remove").into_response(),
    }
}

async fn update_sub_category_handler(
    _session: auth::AuthSession,
    State(state): State<Arc<AppState>>,
    Path(id): Path<i64>,
    Form(form): Form<UpdateSubCategoryForm>,
) -> Response {
    let name = form.category_name.as_deref().map(str::trim).unwrap_or("");
    let category_id = if name.is_empty() {
        None
    } else {
        fdb::get_or_create_category(&state.db, name)
            .ok()
            .map(|c| c.id)
    };
    let _ = fdb::update_subscription_category(&state.db, id, category_id);
    Redirect::to("/admin?success=Category+updated").into_response()
}

async fn add_category_handler(
    _session: auth::AuthSession,
    State(state): State<Arc<AppState>>,
    Form(form): Form<AddCategoryForm>,
) -> Response {
    let name = form.name.trim();
    if name.is_empty() {
        return Redirect::to("/admin?error=Name+required").into_response();
    }
    match fdb::get_or_create_category(&state.db, name) {
        Ok(_) => Redirect::to("/admin?success=Category+added").into_response(),
        Err(_) => Redirect::to("/admin?error=Failed+to+add+category").into_response(),
    }
}

async fn delete_category_handler(
    _session: auth::AuthSession,
    State(state): State<Arc<AppState>>,
    Path(id): Path<i64>,
) -> Response {
    let _ = fdb::delete_category(&state.db, id);
    Redirect::to("/admin?success=Category+removed").into_response()
}

async fn import_opml_handler(
    _session: auth::AuthSession,
    State(state): State<Arc<AppState>>,
    mut multipart: Multipart,
) -> Response {
    let mut content: Option<String> = None;
    while let Ok(Some(field)) = multipart.next_field().await {
        if field.name() == Some("file") {
            if let Ok(s) = field.text().await {
                content = Some(s);
            }
        }
    }
    let Some(content) = content else {
        return Redirect::to("/admin?error=No+file+uploaded").into_response();
    };
    let summary = api::import_opml_str(state, &content).await;
    let msg = format!(
        "Imported+{}%2C+skipped+{}",
        summary.imported, summary.skipped
    );
    Redirect::to(&format!("/admin?success={msg}")).into_response()
}

async fn update_settings_handler(
    _session: auth::AuthSession,
    State(state): State<Arc<AppState>>,
    Form(form): Form<UpdateSettingsForm>,
) -> Response {
    if !(1..=1440).contains(&form.poll_interval_minutes) {
        return Redirect::to("/admin?error=Interval+must+be+1-1440").into_response();
    }
    let _ = fdb::set_setting(
        &state.db,
        POLL_INTERVAL_KEY,
        &form.poll_interval_minutes.to_string(),
    );
    Redirect::to("/admin?success=Settings+saved").into_response()
}

// ── main ──────────────────────────────────────────────────────────────

#[tokio::main]
async fn main() {
    dotenvy::dotenv().ok();
    tracing_subscriber::fmt()
        .with_env_filter(
            tracing_subscriber::EnvFilter::try_from_default_env()
                .unwrap_or_else(|_| tracing_subscriber::EnvFilter::new("info,feeds=info")),
        )
        .init();

    let db_path = std::env::var("DB_PATH").unwrap_or_else(|_| "feeds.sqlite".to_string());
    let conn = Connection::open(&db_path).expect("open sqlite");
    conn.execute_batch(SESSION_SCHEMA).expect("session schema");
    conn.execute_batch(fdb::FEEDS_SCHEMA).expect("feeds schema");
    let db: Db = Arc::new(Mutex::new(conn));

    let cookie_secure = std::env::var("COOKIE_SECURE")
        .map(|v| v.eq_ignore_ascii_case("true"))
        .unwrap_or(false);
    let base_url =
        std::env::var("BASE_URL").unwrap_or_else(|_| "http://localhost:3000".to_string());
    let default_poll_minutes: u64 = std::env::var("DEFAULT_POLL_MINUTES")
        .ok()
        .and_then(|v| v.parse().ok())
        .unwrap_or(30);
    let item_cap: usize = std::env::var("ITEM_CAP_PER_FEED")
        .ok()
        .and_then(|v| v.parse().ok())
        .unwrap_or(200);

    // Seed poll-interval setting if missing so the admin UI shows a value.
    if fdb::get_setting(&db, POLL_INTERVAL_KEY).ok().flatten().is_none() {
        let _ = fdb::set_setting(&db, POLL_INTERVAL_KEY, &default_poll_minutes.to_string());
    }

    let api_key = std::env::var("API_KEY").ok().filter(|s| !s.is_empty());
    if api_key.is_none() {
        tracing::warn!("API_KEY is not set; /api is accessible via session cookie only");
    }

    let state = Arc::new(AppState {
        db,
        admin_password: std::env::var("ADMIN_PASSWORD").ok(),
        api_key,
        cookie_secure,
        base_url,
        default_poll_minutes,
        item_cap,
    });

    tokio::spawn(poller::run(state.clone()));

    let admin_router = Router::new()
        .route("/admin", get(admin_handler))
        .route(
            "/admin/login",
            get(login_get_handler).post(login_post_handler),
        )
        .route("/admin/logout", get(logout_handler))
        .route("/admin/add-feed", post(add_feed_handler))
        .route("/admin/feeds/{id}/delete", post(delete_feed_handler))
        .route("/admin/feeds/{id}/category", post(update_sub_category_handler))
        .route("/admin/categories", post(add_category_handler))
        .route("/admin/categories/{id}/delete", post(delete_category_handler))
        .route("/admin/import-opml", post(import_opml_handler))
        .route("/admin/settings", post(update_settings_handler))
        .route("/admin/discover-feeds", post(discover_feeds_handler));

    let api_router = Router::new()
        .route("/api/items", get(api::list_items))
        .route("/api/items/{id}/read", post(api::mark_item_read))
        .route("/api/items/{id}/unread", post(api::mark_item_unread))
        .route(
            "/api/subscriptions",
            get(api::list_subscriptions).post(api::create_subscription),
        )
        .route(
            "/api/subscriptions/{id}",
            delete(api::delete_subscription).patch(api::update_subscription),
        )
        .route(
            "/api/categories",
            get(api::list_categories).post(api::create_category),
        )
        .route("/api/categories/{id}", delete(api::delete_category))
        .route("/api/import/opml", post(api::import_opml))
        .route(
            "/api/settings",
            get(api::get_settings).put(api::update_settings),
        )
        .route("/api/discover", post(api::discover));

    let app = Router::new()
        .route("/", get(index_handler))
        .route("/feeds.opml", get(feeds_opml_handler))
        .route("/static/{*path}", get(static_handler))
        .merge(admin_router)
        .merge(api_router)
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

    tracing::info!("Feeds server running on http://{host}:{port}");
    axum::serve(listener, app).await.unwrap();
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn escape_xml_all_special() {
        assert_eq!(
            escape_xml(r#"<a href="x">&'test'</a>"#),
            "&lt;a href=&quot;x&quot;&gt;&amp;&apos;test&apos;&lt;/a&gt;"
        );
    }

    #[test]
    fn format_date_valid_timestamp() {
        assert_eq!(format_date(1705276800), "Jan 15, 2024");
    }
}
