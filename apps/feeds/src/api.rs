use std::sync::Arc;

use andromeda_db::feeds as fdb;
use axum::{
    extract::{Multipart, Path, Query, State},
    http::StatusCode,
    response::{IntoResponse, Response},
    Json,
};
use serde::Deserialize;

use andromeda_db::Db;

use crate::auth::ApiAuth;
use crate::feeds::{discover_favicon, discover_feeds, fetch_feed, parse_opml, ParsedEntry};
use crate::poller::POLL_INTERVAL_KEY;
use crate::server::AppState;

fn err_json(status: StatusCode, msg: impl Into<String>) -> Response {
    (
        status,
        Json(serde_json::json!({ "error": msg.into() })),
    )
        .into_response()
}

// ── Items ─────────────────────────────────────────────────────────────

#[derive(Deserialize)]
pub struct ListItemsQuery {
    limit: Option<i64>,
    #[serde(default)]
    unread: bool,
    category_id: Option<i64>,
    subscription_id: Option<i64>,
}

pub async fn list_items(
    State(state): State<Arc<AppState>>,
    Query(q): Query<ListItemsQuery>,
) -> Response {
    let filter = fdb::ListItemsFilter {
        limit: q.limit,
        unread_only: q.unread,
        category_id: q.category_id,
        subscription_id: q.subscription_id,
    };
    match fdb::list_items(&state.db, &filter) {
        Ok(items) => Json(serde_json::json!({ "items": items })).into_response(),
        Err(e) => err_json(StatusCode::INTERNAL_SERVER_ERROR, e.to_string()),
    }
}

pub async fn mark_item_read(
    _auth: ApiAuth,
    State(state): State<Arc<AppState>>,
    Path(id): Path<i64>,
) -> Response {
    match fdb::mark_read(&state.db, id) {
        Ok(true) => Json(serde_json::json!({ "ok": true, "is_read": true })).into_response(),
        Ok(false) => err_json(StatusCode::NOT_FOUND, "item not found"),
        Err(e) => err_json(StatusCode::INTERNAL_SERVER_ERROR, e.to_string()),
    }
}

pub async fn mark_item_unread(
    _auth: ApiAuth,
    State(state): State<Arc<AppState>>,
    Path(id): Path<i64>,
) -> Response {
    match fdb::mark_unread(&state.db, id) {
        Ok(true) => Json(serde_json::json!({ "ok": true, "is_read": false })).into_response(),
        Ok(false) => err_json(StatusCode::NOT_FOUND, "item not found"),
        Err(e) => err_json(StatusCode::INTERNAL_SERVER_ERROR, e.to_string()),
    }
}

// ── Subscriptions ─────────────────────────────────────────────────────

pub async fn list_subscriptions(
    State(state): State<Arc<AppState>>,
) -> Response {
    match fdb::list_subscriptions(&state.db) {
        Ok(subs) => Json(serde_json::json!({ "subscriptions": subs })).into_response(),
        Err(e) => err_json(StatusCode::INTERNAL_SERVER_ERROR, e.to_string()),
    }
}

#[derive(Deserialize)]
pub struct CreateSubscriptionBody {
    pub feed_url: String,
    pub title: Option<String>,
    pub category_id: Option<i64>,
    pub category_name: Option<String>,
}

pub async fn create_subscription(
    _auth: ApiAuth,
    State(state): State<Arc<AppState>>,
    Json(body): Json<CreateSubscriptionBody>,
) -> Response {
    add_subscription(&state, &body).await
}

pub async fn add_subscription(
    state: &AppState,
    body: &CreateSubscriptionBody,
) -> Response {
    let feed_url = body.feed_url.trim();
    if feed_url.is_empty() {
        return err_json(StatusCode::BAD_REQUEST, "feed_url required");
    }

    if let Ok(Some(existing)) = fdb::get_subscription_by_url(&state.db, feed_url) {
        return (
            StatusCode::CONFLICT,
            Json(serde_json::json!({
                "error": "already subscribed",
                "subscription": existing
            })),
        )
            .into_response();
    }

    // Probe once to resolve title + site_url + initial entries.
    let probed = fetch_feed(feed_url, None, None).await;
    let (title, site_url, etag, last_modified, entries) = match probed {
        Ok(r) => (
            body.title
                .clone()
                .or(r.title)
                .unwrap_or_else(|| feed_url.to_string()),
            r.site_url,
            r.etag,
            r.last_modified,
            r.entries,
        ),
        Err(e) => {
            return err_json(
                StatusCode::BAD_REQUEST,
                format!("feed not reachable: {e}"),
            );
        }
    };

    let category_id = match resolve_category(state, body.category_id, body.category_name.as_deref())
    {
        Ok(id) => id,
        Err(resp) => return resp,
    };

    let mut sub = match fdb::insert_subscription(
        &state.db,
        feed_url,
        &title,
        site_url.as_deref(),
        category_id,
    ) {
        Ok(s) => s,
        Err(e) => return err_json(StatusCode::INTERNAL_SERVER_ERROR, e.to_string()),
    };

    if let Some(site) = site_url.as_deref() {
        if let Some(fav) = discover_favicon(site).await {
            let _ = fdb::update_subscription_favicon(&state.db, sub.id, Some(&fav));
            sub.favicon_url = Some(fav);
        }
    }

    seed_subscription(
        &state.db,
        sub.id,
        &entries,
        etag.as_deref(),
        last_modified.as_deref(),
        state.item_cap,
    );

    (StatusCode::CREATED, Json(serde_json::json!({ "subscription": sub }))).into_response()
}

/// Insert probe entries into the new subscription, prune to the item cap, then
/// persist etag/last_modified. The order matters: persisting the conditional-fetch
/// metadata before seeding would let the next poller pass receive a 304 against an
/// empty subscription, leaving it permanently dry until upstream changes.
pub(crate) fn seed_subscription(
    db: &Db,
    sub_id: i64,
    entries: &[ParsedEntry],
    etag: Option<&str>,
    last_modified: Option<&str>,
    item_cap: usize,
) -> usize {
    let mut inserted = 0usize;
    for entry in entries {
        if entry.link.is_empty() {
            continue;
        }
        match fdb::insert_item_ignore_dup(
            db,
            &fdb::NewItem {
                subscription_id: sub_id,
                guid: &entry.guid,
                title: &entry.title,
                link: &entry.link,
                author: entry.author.as_deref(),
                published_at: entry.published_at,
            },
        ) {
            Ok(true) => inserted += 1,
            Ok(false) => {}
            Err(e) => tracing::warn!("seed insert failed for sub {sub_id}: {e}"),
        }
    }
    let _ = fdb::prune_subscription(db, sub_id, item_cap as i64);
    let _ = fdb::update_subscription_meta(
        db,
        sub_id,
        etag,
        last_modified,
        &chrono::Utc::now().format("%Y-%m-%d %H:%M:%S").to_string(),
        None,
    );
    inserted
}

fn resolve_category(
    state: &AppState,
    id: Option<i64>,
    name: Option<&str>,
) -> Result<Option<i64>, Response> {
    if let Some(id) = id {
        return Ok(Some(id));
    }
    if let Some(raw) = name {
        let trimmed = raw.trim();
        if !trimmed.is_empty() {
            let cat = fdb::get_or_create_category(&state.db, trimmed)
                .map_err(|e| err_json(StatusCode::INTERNAL_SERVER_ERROR, e.to_string()))?;
            return Ok(Some(cat.id));
        }
    }
    Ok(None)
}

#[derive(Deserialize)]
pub struct UpdateSubscriptionBody {
    category_id: Option<i64>,
    category_name: Option<String>,
    clear_category: Option<bool>,
}

pub async fn update_subscription(
    _auth: ApiAuth,
    State(state): State<Arc<AppState>>,
    Path(id): Path<i64>,
    Json(body): Json<UpdateSubscriptionBody>,
) -> Response {
    let category_id = if body.clear_category.unwrap_or(false) {
        None
    } else {
        match resolve_category(&state, body.category_id, body.category_name.as_deref()) {
            Ok(v) => v,
            Err(resp) => return resp,
        }
    };

    match fdb::update_subscription_category(&state.db, id, category_id) {
        Ok(()) => Json(serde_json::json!({ "ok": true })).into_response(),
        Err(e) => err_json(StatusCode::INTERNAL_SERVER_ERROR, e.to_string()),
    }
}

pub async fn delete_subscription(
    _auth: ApiAuth,
    State(state): State<Arc<AppState>>,
    Path(id): Path<i64>,
) -> Response {
    match fdb::delete_subscription(&state.db, id) {
        Ok(true) => Json(serde_json::json!({ "ok": true })).into_response(),
        Ok(false) => err_json(StatusCode::NOT_FOUND, "subscription not found"),
        Err(e) => err_json(StatusCode::INTERNAL_SERVER_ERROR, e.to_string()),
    }
}

// ── Categories ────────────────────────────────────────────────────────

pub async fn list_categories(
    _auth: ApiAuth,
    State(state): State<Arc<AppState>>,
) -> Response {
    match fdb::list_categories(&state.db) {
        Ok(cats) => Json(serde_json::json!({ "categories": cats })).into_response(),
        Err(e) => err_json(StatusCode::INTERNAL_SERVER_ERROR, e.to_string()),
    }
}

#[derive(Deserialize)]
pub struct CreateCategoryBody {
    name: String,
}

pub async fn create_category(
    _auth: ApiAuth,
    State(state): State<Arc<AppState>>,
    Json(body): Json<CreateCategoryBody>,
) -> Response {
    let name = body.name.trim();
    if name.is_empty() {
        return err_json(StatusCode::BAD_REQUEST, "name required");
    }
    match fdb::get_or_create_category(&state.db, name) {
        Ok(cat) => (StatusCode::CREATED, Json(serde_json::json!({ "category": cat }))).into_response(),
        Err(e) => err_json(StatusCode::INTERNAL_SERVER_ERROR, e.to_string()),
    }
}

pub async fn delete_category(
    _auth: ApiAuth,
    State(state): State<Arc<AppState>>,
    Path(id): Path<i64>,
) -> Response {
    match fdb::delete_category(&state.db, id) {
        Ok(true) => Json(serde_json::json!({ "ok": true })).into_response(),
        Ok(false) => err_json(StatusCode::NOT_FOUND, "category not found"),
        Err(e) => err_json(StatusCode::INTERNAL_SERVER_ERROR, e.to_string()),
    }
}

// ── OPML import ───────────────────────────────────────────────────────

pub async fn import_opml(
    _auth: ApiAuth,
    State(state): State<Arc<AppState>>,
    mut multipart: Multipart,
) -> Response {
    let mut content: Option<String> = None;
    while let Ok(Some(field)) = multipart.next_field().await {
        if field.name() == Some("file") {
            match field.text().await {
                Ok(s) => content = Some(s),
                Err(e) => {
                    return err_json(StatusCode::BAD_REQUEST, format!("read file failed: {e}"));
                }
            }
        }
    }

    let content = match content {
        Some(c) => c,
        None => return err_json(StatusCode::BAD_REQUEST, "missing `file` field"),
    };

    let result = import_opml_str(state, &content).await;
    Json(serde_json::json!(result)).into_response()
}

#[derive(serde::Serialize)]
pub struct ImportSummary {
    pub imported: usize,
    pub skipped: usize,
    pub failed: Vec<String>,
}

const SEED_CONCURRENCY: usize = 8;

pub async fn import_opml_str(state: Arc<AppState>, content: &str) -> ImportSummary {
    let entries = parse_opml(content);
    let mut imported = 0usize;
    let mut skipped = 0usize;
    let mut failed: Vec<String> = Vec::new();
    let sem = Arc::new(tokio::sync::Semaphore::new(SEED_CONCURRENCY));
    let mut seed_handles: Vec<tokio::task::JoinHandle<Option<String>>> = Vec::new();

    for entry in entries {
        if let Ok(Some(_)) = fdb::get_subscription_by_url(&state.db, &entry.xml_url) {
            skipped += 1;
            continue;
        }

        let category_id = entry
            .category
            .as_deref()
            .and_then(|name| fdb::get_or_create_category(&state.db, name).ok())
            .map(|c| c.id);

        let title = entry
            .title
            .clone()
            .unwrap_or_else(|| entry.xml_url.clone());
        let site_url = entry.html_url.clone();

        match fdb::insert_subscription(
            &state.db,
            &entry.xml_url,
            &title,
            site_url.as_deref(),
            category_id,
        ) {
            Ok(sub) => {
                imported += 1;
                let state_cloned = Arc::clone(&state);
                let sem_cloned = Arc::clone(&sem);
                let site = site_url.clone();
                seed_handles.push(tokio::spawn(async move {
                    let _permit = match sem_cloned.acquire().await {
                        Ok(p) => p,
                        Err(_) => return None,
                    };
                    if let Some(site) = site.as_deref() {
                        if let Some(fav) = discover_favicon(site).await {
                            let _ = fdb::update_subscription_favicon(
                                &state_cloned.db,
                                sub.id,
                                Some(&fav),
                            );
                        }
                    }
                    crate::poller::poll_one(&state_cloned, &sub)
                        .await
                        .err()
                        .map(|e| format!("{}: seed failed: {}", sub.feed_url, e))
                }));
            }
            Err(e) => failed.push(format!("{}: {}", entry.xml_url, e)),
        }
    }

    for h in seed_handles {
        if let Ok(Some(msg)) = h.await {
            failed.push(msg);
        }
    }

    ImportSummary {
        imported,
        skipped,
        failed,
    }
}

// ── Settings ──────────────────────────────────────────────────────────

#[derive(serde::Serialize)]
struct SettingsView {
    poll_interval_minutes: u64,
    default_poll_minutes: u64,
    item_cap_per_feed: usize,
    api_key_configured: bool,
}

pub async fn get_settings(
    _auth: ApiAuth,
    State(state): State<Arc<AppState>>,
) -> Response {
    let poll = fdb::get_setting(&state.db, POLL_INTERVAL_KEY)
        .ok()
        .flatten()
        .and_then(|v| v.parse::<u64>().ok())
        .unwrap_or(state.default_poll_minutes);
    let view = SettingsView {
        poll_interval_minutes: poll,
        default_poll_minutes: state.default_poll_minutes,
        item_cap_per_feed: state.item_cap,
        api_key_configured: state.api_key.is_some(),
    };
    Json(serde_json::json!(view)).into_response()
}

#[derive(Deserialize)]
pub struct UpdateSettingsBody {
    poll_interval_minutes: Option<u64>,
}

pub async fn update_settings(
    _auth: ApiAuth,
    State(state): State<Arc<AppState>>,
    Json(body): Json<UpdateSettingsBody>,
) -> Response {
    if let Some(mins) = body.poll_interval_minutes {
        if !(1..=1440).contains(&mins) {
            return err_json(
                StatusCode::BAD_REQUEST,
                "poll_interval_minutes must be between 1 and 1440",
            );
        }
        if let Err(e) = fdb::set_setting(&state.db, POLL_INTERVAL_KEY, &mins.to_string()) {
            return err_json(StatusCode::INTERNAL_SERVER_ERROR, e.to_string());
        }
    }
    Json(serde_json::json!({ "ok": true })).into_response()
}

// ── Preview ───────────────────────────────────────────────────────────

#[derive(Deserialize)]
pub struct PreviewQuery {
    url: Option<String>,
    urls: Option<String>,
}

pub async fn preview(Query(q): Query<PreviewQuery>) -> Response {
    let raw = match q.url.or(q.urls) {
        Some(s) => s,
        None => return err_json(StatusCode::BAD_REQUEST, "url required"),
    };
    let urls: Vec<String> = raw
        .split(',')
        .map(|u| u.trim().to_string())
        .filter(|u| !u.is_empty())
        .collect();
    if urls.is_empty() {
        return err_json(StatusCode::BAD_REQUEST, "no URLs provided");
    }
    let items = crate::feeds::preview_urls(&urls).await;
    Json(serde_json::json!({ "items": items })).into_response()
}

// ── Discover ──────────────────────────────────────────────────────────

#[derive(Deserialize)]
pub struct DiscoverBody {
    base_url: String,
}

pub async fn discover(
    _auth: ApiAuth,
    Json(body): Json<DiscoverBody>,
) -> Response {
    match discover_feeds(&body.base_url).await {
        Ok(feeds) => Json(serde_json::json!({ "feeds": feeds })).into_response(),
        Err(e) => err_json(StatusCode::BAD_REQUEST, e),
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use andromeda_db::feeds::{
        get_subscription, insert_subscription, list_items, ListItemsFilter, FEEDS_SCHEMA,
    };
    use rusqlite::Connection;
    use std::sync::Mutex;

    fn test_db() -> Db {
        let conn = Connection::open_in_memory().unwrap();
        conn.execute_batch(FEEDS_SCHEMA).unwrap();
        Arc::new(Mutex::new(conn))
    }

    fn entry(guid: &str, link: &str, ts: i64) -> ParsedEntry {
        ParsedEntry {
            guid: guid.into(),
            title: format!("post {guid}"),
            link: link.into(),
            author: None,
            published_at: ts,
        }
    }

    #[test]
    fn seed_subscription_inserts_entries_and_persists_meta() {
        let db = test_db();
        let sub = insert_subscription(&db, "https://x.com/feed", "X", None, None).unwrap();
        let entries = vec![
            entry("g1", "https://x.com/1", 100),
            entry("g2", "https://x.com/2", 200),
        ];

        let inserted =
            seed_subscription(&db, sub.id, &entries, Some("etag-1"), Some("Sun, 01 Jan"), 50);

        assert_eq!(inserted, 2);
        let items = list_items(&db, &ListItemsFilter::default()).unwrap();
        assert_eq!(items.len(), 2);
        let after = get_subscription(&db, sub.id).unwrap().unwrap();
        assert_eq!(after.etag.as_deref(), Some("etag-1"));
        assert_eq!(after.last_modified.as_deref(), Some("Sun, 01 Jan"));
        assert!(after.last_fetched_at.is_some());
        assert!(after.last_error.is_none());
    }

    #[test]
    fn seed_subscription_skips_empty_links() {
        let db = test_db();
        let sub = insert_subscription(&db, "https://x.com/feed", "X", None, None).unwrap();
        let entries = vec![entry("g1", "", 100), entry("g2", "https://x.com/2", 200)];

        let inserted = seed_subscription(&db, sub.id, &entries, None, None, 50);

        assert_eq!(inserted, 1);
        let items = list_items(&db, &ListItemsFilter::default()).unwrap();
        assert_eq!(items.len(), 1);
        assert_eq!(items[0].guid, "g2");
    }

    #[test]
    fn seed_subscription_dedups_on_repeat() {
        let db = test_db();
        let sub = insert_subscription(&db, "https://x.com/feed", "X", None, None).unwrap();
        let entries = vec![entry("g1", "https://x.com/1", 100)];

        assert_eq!(seed_subscription(&db, sub.id, &entries, None, None, 50), 1);
        assert_eq!(seed_subscription(&db, sub.id, &entries, None, None, 50), 0);
        assert_eq!(list_items(&db, &ListItemsFilter::default()).unwrap().len(), 1);
    }

    #[test]
    fn seed_subscription_prunes_to_item_cap() {
        let db = test_db();
        let sub = insert_subscription(&db, "https://x.com/feed", "X", None, None).unwrap();
        let entries: Vec<_> = (0..10)
            .map(|i| entry(&format!("g{i}"), &format!("https://x.com/{i}"), i as i64))
            .collect();

        seed_subscription(&db, sub.id, &entries, None, None, 3);

        let items = list_items(&db, &ListItemsFilter::default()).unwrap();
        assert_eq!(items.len(), 3);
        // newest survive
        assert_eq!(items[0].published_at, 9);
        assert_eq!(items[2].published_at, 7);
    }

    #[test]
    fn seed_subscription_with_no_entries_still_persists_meta() {
        let db = test_db();
        let sub = insert_subscription(&db, "https://x.com/feed", "X", None, None).unwrap();

        let inserted = seed_subscription(&db, sub.id, &[], Some("etag-empty"), None, 50);

        assert_eq!(inserted, 0);
        let after = get_subscription(&db, sub.id).unwrap().unwrap();
        assert_eq!(after.etag.as_deref(), Some("etag-empty"));
    }
}
