mod auth;
mod feeds;
mod models;

use askama::Template;
use axum::{
    extract::{Query, State},
    http::{header, HeaderMap, StatusCode},
    response::{Html, IntoResponse, Json, Redirect, Response},
    routing::{get, post},
    Form, Router,
};
use chrono::DateTime;
use rust_embed::Embed;
use serde::Deserialize;
use std::collections::HashMap;
use std::sync::Arc;

#[derive(Embed)]
#[folder = "static/"]
struct Static;

pub struct AppState {
    sessions: auth::SessionStore,
    admin_password: Option<String>,
    cookie_secure: bool,
    base_url: String,
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
    freshrss_configured: bool,
    success: Option<String>,
    error: Option<String>,
    subscriptions: Option<Vec<models::Subscription>>,
}

fn format_date(timestamp: i64) -> String {
    DateTime::from_timestamp(timestamp, 0)
        .map(|dt| dt.format("%b %-d, %Y").to_string())
        .unwrap_or_default()
}

fn freshrss_env() -> Option<(String, String, String)> {
    let url = std::env::var("FRESHRSS_URL").ok()?;
    let username = std::env::var("FRESHRSS_USERNAME").ok()?;
    let password = std::env::var("FRESHRSS_PASSWORD").ok()?;
    Some((url, username, password))
}

async fn index_handler(
    State(state): State<Arc<AppState>>,
    Query(params): Query<HashMap<String, String>>,
) -> impl IntoResponse {
    let url_query = params
        .get("url")
        .or_else(|| params.get("urls"))
        .map(|s| s.as_str());

    let template = match feeds::get_feed_items(url_query).await {
        Ok((items, feed_urls)) => {
            let template_items: Vec<TemplateFeedItem> = items
                .into_iter()
                .map(|item| TemplateFeedItem {
                    title: item.title,
                    link: item.link,
                    author: item.author,
                    formatted_date: format_date(item.published),
                })
                .collect();

            IndexTemplate {
                base_url: state.base_url.clone(),
                items: template_items,
                feed_urls,
                error: None,
            }
        }
        Err(e) => {
            eprintln!("Error fetching feeds: {e}");
            IndexTemplate {
                base_url: state.base_url.clone(),
                items: Vec::new(),
                feed_urls: None,
                error: Some("Error loading feeds. Please try again later.".to_string()),
            }
        }
    };

    Html(template.render().unwrap())
}

async fn feeds_handler(
    Query(params): Query<HashMap<String, String>>,
) -> Result<Response, StatusCode> {
    let format = params
        .get("format")
        .map(|s| s.as_str())
        .unwrap_or("json");

    let freshrss_url =
        std::env::var("FRESHRSS_URL").map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?;
    let username =
        std::env::var("FRESHRSS_USERNAME").map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?;
    let password =
        std::env::var("FRESHRSS_PASSWORD").map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?;

    let data = feeds::fetch_freshrss_subscriptions(&freshrss_url, &username, &password)
        .await
        .map_err(|e| {
            eprintln!("Failed to fetch subscriptions: {e}");
            StatusCode::INTERNAL_SERVER_ERROR
        })?;

    match format {
        "json" => Ok(Json(serde_json::json!(data)).into_response()),
        "opml" => {
            let now = chrono::Utc::now().to_rfc2822();
            let subscriptions = data.subscriptions.unwrap_or_default();

            let mut opml = format!(
                r#"<?xml version="1.0" encoding="UTF-8"?>
<opml version="2.0">
  <head>
    <title>Steve's Feeds</title>
    <dateCreated>{now}</dateCreated>
  </head>
  <body>
"#
            );

            for feed in &subscriptions {
                opml.push_str(&format!(
                    "    <outline type=\"rss\" text=\"{}\" title=\"{}\" xmlUrl=\"{}\" htmlUrl=\"{}\" />\n",
                    escape_xml(&feed.title),
                    escape_xml(&feed.title),
                    escape_xml(&feed.url),
                    escape_xml(feed.html_url.as_deref().unwrap_or("")),
                ));
            }

            opml.push_str("  </body>\n</opml>");

            Ok((
                [
                    (header::CONTENT_TYPE, "application/xml"),
                    (
                        header::CONTENT_DISPOSITION,
                        "attachment; filename=\"feeds.opml\"",
                    ),
                ],
                opml,
            )
                .into_response())
        }
        _ => Ok((
            StatusCode::BAD_REQUEST,
            Json(serde_json::json!({
                "error": "Invalid format. Use ?format=json or ?format=opml"
            })),
        )
            .into_response()),
    }
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
            (
                [(header::CONTENT_TYPE, mime.as_ref())],
                file.data.to_vec(),
            )
                .into_response()
        }
        None => StatusCode::NOT_FOUND.into_response(),
    }
}

// --- Admin routes ---

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
}

async fn login_get_handler(Query(q): Query<FlashQuery>) -> impl IntoResponse {
    Html(LoginTemplate { error: q.error }.render().unwrap())
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
    auth::create_session(&state.sessions, &token);
    let cookie = auth::build_session_cookie(&token, state.cookie_secure);

    let mut resp = Redirect::to("/admin").into_response();
    resp.headers_mut()
        .insert(header::SET_COOKIE, cookie.parse().unwrap());
    resp
}

async fn logout_handler(
    State(state): State<Arc<AppState>>,
    headers: HeaderMap,
) -> Response {
    if let Some(token) = auth::extract_session_cookie(&headers) {
        auth::delete_session(&state.sessions, &token);
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
    let _ = state; // state available if needed later

    let freshrss_configured = freshrss_env().is_some();

    let subscriptions = if freshrss_configured {
        if let Some((url, user, pass)) = freshrss_env() {
            feeds::fetch_freshrss_subscriptions(&url, &user, &pass)
                .await
                .ok()
                .and_then(|list| list.subscriptions)
        } else {
            None
        }
    } else {
        None
    };

    Html(
        AdminTemplate {
            freshrss_configured,
            success: q.success,
            error: q.error,
            subscriptions,
        }
        .render()
        .unwrap(),
    )
    .into_response()
}

async fn add_feed_handler(
    _session: auth::AuthSession,
    Form(form): Form<AddFeedForm>,
) -> Response {
    let (url, user, pass) = match freshrss_env() {
        Some(env) => env,
        None => {
            return Redirect::to("/admin?error=FreshRSS+not+configured").into_response();
        }
    };

    match feeds::add_freshrss_subscription(&url, &user, &pass, &form.feed_url).await {
        Ok(_) => Redirect::to("/admin?success=Feed+added+successfully").into_response(),
        Err(e) => {
            eprintln!("Failed to add feed: {e}");
            let encoded = urlencoding::encode(&e);
            Redirect::to(&format!("/admin?error={encoded}")).into_response()
        }
    }
}

#[tokio::main]
async fn main() {
    dotenvy::dotenv().ok();

    let cookie_secure = std::env::var("COOKIE_SECURE")
        .map(|v| v.eq_ignore_ascii_case("true"))
        .unwrap_or(false);

    let base_url = std::env::var("BASE_URL").unwrap_or_else(|_| "http://localhost:3000".to_string());

    let state = Arc::new(AppState {
        sessions: auth::new_session_store(),
        admin_password: std::env::var("ADMIN_PASSWORD").ok(),
        cookie_secure,
        base_url,
    });

    let app = Router::new()
        .route("/", get(index_handler))
        .route("/feeds", get(feeds_handler))
        .route("/admin", get(admin_handler))
        .route(
            "/admin/login",
            get(login_get_handler).post(login_post_handler),
        )
        .route("/admin/logout", get(logout_handler))
        .route("/admin/add-feed", post(add_feed_handler))
        .route("/static/{*path}", get(static_handler))
        .with_state(state);

    let host = std::env::var("HOST").unwrap_or_else(|_| "0.0.0.0".to_string());
    let port: u16 = std::env::var("PORT")
        .ok()
        .and_then(|v| v.parse().ok())
        .unwrap_or(3000);
    let addr = format!("{}:{}", host, port);
    let listener = tokio::net::TcpListener::bind(&addr)
        .await
        .unwrap_or_else(|_| panic!("Failed to bind to {}", addr));

    println!("Server running on http://{}:{}", host, port);
    axum::serve(listener, app).await.unwrap();
}
