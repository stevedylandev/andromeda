use std::sync::Arc;
use std::time::Duration;

use chrono::{Duration as ChronoDuration, NaiveDate, TimeZone, Utc};
use chrono_tz::Tz;

use crate::aic;
use crate::db::{self, DailyArtwork};
use crate::server::AppState;

pub async fn run(state: Arc<AppState>) {
    if let Err(e) = ensure_day(&state, &today_in_tz(&state.tz)).await {
        tracing::warn!("startup ensure_day failed: {e}");
    }

    if state.backfill_days > 0 {
        let dates = past_n_dates(&state.tz, state.backfill_days);
        match db::missing_dates(&state.db, &dates) {
            Ok(missing) => {
                tracing::info!(
                    "backfill: {} of last {} days missing",
                    missing.len(),
                    state.backfill_days
                );
                for d in missing {
                    if let Err(e) = ensure_day(&state, &d).await {
                        tracing::warn!("backfill {d} failed: {e}");
                    }
                }
            }
            Err(e) => tracing::error!("backfill missing_dates failed: {e}"),
        }
    }

    loop {
        let dur = duration_until_next_midnight(&state.tz);
        tracing::info!(
            "scheduler sleeping for {}s until next midnight in {}",
            dur.as_secs(),
            state.tz.name()
        );
        tokio::time::sleep(dur).await;
        let date = today_in_tz(&state.tz);
        if let Err(e) = ensure_day(&state, &date).await {
            tracing::warn!("scheduled ensure_day {date} failed: {e}");
        }
    }
}

pub async fn ensure_day(state: &AppState, date: &str) -> Result<(), String> {
    if db::get_daily(&state.db, date)
        .map_err(|e| e.to_string())?
        .is_some()
    {
        return Ok(());
    }
    let raw = aic::pick_unique(
        &state.http,
        &state.db,
        &state.classifications,
        state.max_dedup_retries,
    )
    .await?;
    let now = Utc::now().to_rfc3339();
    let daily: DailyArtwork = aic::raw_to_daily(raw, date.to_string(), now)
        .ok_or_else(|| "missing image_id on selected artwork".to_string())?;
    db::insert_daily(&state.db, &daily).map_err(|e| e.to_string())?;
    tracing::info!(
        "stored artwork {} for {} (image_id={})",
        daily.artwork_id,
        date,
        daily.image_id
    );
    Ok(())
}

pub fn today_in_tz(tz: &Tz) -> String {
    Utc::now()
        .with_timezone(tz)
        .date_naive()
        .format("%Y-%m-%d")
        .to_string()
}

pub fn past_n_dates(tz: &Tz, n: u32) -> Vec<String> {
    let today = Utc::now().with_timezone(tz).date_naive();
    (1..=n as i64)
        .filter_map(|i| today.checked_sub_signed(ChronoDuration::days(i)))
        .map(|d| d.format("%Y-%m-%d").to_string())
        .collect()
}

pub fn parse_date(s: &str) -> Option<NaiveDate> {
    NaiveDate::parse_from_str(s, "%Y-%m-%d").ok()
}

fn duration_until_next_midnight(tz: &Tz) -> Duration {
    let now = Utc::now().with_timezone(tz);
    let next_day = now.date_naive() + ChronoDuration::days(1);
    let next_midnight = tz
        .from_local_datetime(&next_day.and_hms_opt(0, 0, 1).expect("valid time"))
        .single()
        .or_else(|| {
            tz.from_local_datetime(&next_day.and_hms_opt(0, 0, 1).expect("valid time"))
                .earliest()
        })
        .unwrap_or_else(|| now + ChronoDuration::days(1));
    let delta = next_midnight.signed_duration_since(now);
    delta
        .to_std()
        .unwrap_or_else(|_| Duration::from_secs(60 * 60))
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn past_n_dates_count() {
        let tz: Tz = "UTC".parse().unwrap();
        let dates = past_n_dates(&tz, 5);
        assert_eq!(dates.len(), 5);
    }

    #[test]
    fn past_n_dates_excludes_today() {
        let tz: Tz = "UTC".parse().unwrap();
        let today = today_in_tz(&tz);
        let dates = past_n_dates(&tz, 3);
        assert!(!dates.contains(&today));
    }

    #[test]
    fn parse_date_valid_invalid() {
        assert!(parse_date("2024-05-01").is_some());
        assert!(parse_date("2024-13-01").is_none());
        assert!(parse_date("notadate").is_none());
    }
}
