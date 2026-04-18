use std::sync::Arc;
use std::time::Duration;

use andromeda_db::feeds as fdb;
use chrono::Utc;

use crate::feeds::{fetch_feed, FetchResult};
use crate::AppState;

pub const POLL_INTERVAL_KEY: &str = "poll_interval_minutes";

pub async fn run(state: Arc<AppState>) {
    // Stagger the first pass so startup is fast.
    tokio::time::sleep(Duration::from_secs(3)).await;
    loop {
        let minutes = poll_interval_minutes(&state);
        tracing::info!("poller pass starting (interval {minutes}m)");
        if let Err(e) = sweep(&state).await {
            tracing::error!("poller sweep failed: {e}");
        }
        tokio::time::sleep(Duration::from_secs(minutes * 60)).await;
    }
}

fn poll_interval_minutes(state: &AppState) -> u64 {
    fdb::get_setting(&state.db, POLL_INTERVAL_KEY)
        .ok()
        .flatten()
        .and_then(|v| v.parse::<u64>().ok())
        .filter(|v| *v >= 1)
        .unwrap_or(state.default_poll_minutes)
}

async fn sweep(state: &AppState) -> Result<(), String> {
    let subs = fdb::list_subscriptions(&state.db).map_err(|e| e.to_string())?;
    for sub in subs {
        if let Err(e) = poll_one(state, &sub).await {
            tracing::warn!("feed {} failed: {}", sub.feed_url, e);
            let now = Utc::now().format("%Y-%m-%d %H:%M:%S").to_string();
            let _ = fdb::update_subscription_meta(
                &state.db,
                sub.id,
                sub.etag.as_deref(),
                sub.last_modified.as_deref(),
                &now,
                Some(&e),
            );
        }
    }
    Ok(())
}

pub async fn poll_one(state: &AppState, sub: &fdb::Subscription) -> Result<usize, String> {
    let result: FetchResult =
        fetch_feed(&sub.feed_url, sub.etag.as_deref(), sub.last_modified.as_deref()).await?;
    let now = Utc::now().format("%Y-%m-%d %H:%M:%S").to_string();

    let mut inserted = 0usize;
    if result.status != 304 {
        for entry in &result.entries {
            if entry.link.is_empty() {
                continue;
            }
            let item = fdb::NewItem {
                subscription_id: sub.id,
                guid: &entry.guid,
                title: &entry.title,
                link: &entry.link,
                author: entry.author.as_deref(),
                published_at: entry.published_at,
            };
            match fdb::insert_item_ignore_dup(&state.db, &item) {
                Ok(true) => inserted += 1,
                Ok(false) => {}
                Err(e) => tracing::warn!("insert item failed for {}: {}", sub.feed_url, e),
            }
        }

        // Refresh title if feed advertises a new one and current title looks placeholder.
        if let Some(new_title) = result.title.as_deref() {
            if !new_title.is_empty() && sub.title != new_title && sub.title == sub.feed_url {
                let _ = fdb::update_subscription_title(&state.db, sub.id, new_title);
            }
        }

        let _ = fdb::prune_subscription(&state.db, sub.id, state.item_cap as i64);
    }

    fdb::update_subscription_meta(
        &state.db,
        sub.id,
        result.etag.as_deref(),
        result.last_modified.as_deref(),
        &now,
        None,
    )
    .map_err(|e| e.to_string())?;

    tracing::info!(
        "{} status={} new={} total_entries={}",
        sub.feed_url,
        result.status,
        inserted,
        result.entries.len()
    );
    Ok(inserted)
}
