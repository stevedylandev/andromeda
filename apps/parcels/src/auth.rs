use axum::{
    extract::{FromRef, FromRequestParts},
    http::request::Parts,
    response::{IntoResponse, Redirect, Response},
};
use std::sync::Arc;

use crate::AppState;

pub use andromeda_auth::{
    build_session_cookie, clear_session_cookie, extract_session_cookie, generate_session_token,
    verify_password,
};

// ── Session Token ──────────────────────────────────────────────────────────

/// Return an ISO datetime string 7 days from now.
pub fn session_expiry_at() -> String {
    use std::time::{SystemTime, UNIX_EPOCH};
    let secs = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .unwrap()
        .as_secs()
        + 7 * 24 * 3600;
    let dt = secs;
    let s = dt % 60;
    let m = (dt / 60) % 60;
    let h = (dt / 3600) % 24;
    let days_since_epoch = dt / 86400;
    format_unix_to_datetime(days_since_epoch, h, m, s)
}

fn format_unix_to_datetime(days: u64, h: u64, m: u64, s: u64) -> String {
    // https://howardhinnant.github.io/date_algorithms.html
    let z = days as i64 + 719468;
    let era = if z >= 0 { z } else { z - 146096 } / 146097;
    let doe = (z - era * 146097) as u64;
    let yoe = (doe - doe / 1460 + doe / 36524 - doe / 146096) / 365;
    let y = yoe as i64 + era * 400;
    let doy = doe - (365 * yoe + yoe / 4 - yoe / 100);
    let mp = (5 * doy + 2) / 153;
    let d = doy - (153 * mp + 2) / 5 + 1;
    let mo = if mp < 10 { mp + 3 } else { mp - 9 };
    let y = if mo <= 2 { y + 1 } else { y };
    format!("{:04}-{:02}-{:02} {:02}:{:02}:{:02}", y, mo, d, h, m, s)
}

pub fn format_unix_to_datetime_pub(days: u64, h: u64, m: u64, s: u64) -> String {
    format_unix_to_datetime(days, h, m, s)
}

pub fn extract_session_token(headers: &axum::http::HeaderMap) -> Option<String> {
    extract_session_cookie(headers)
}

// ── Axum Extractor ─────────────────────────────────────────────────────────

/// Authenticated session guard. Extract from request; redirects to /login if not valid.
pub struct AuthSession;

impl<S> FromRequestParts<S> for AuthSession
where
    S: Send + Sync,
    Arc<AppState>: FromRef<S>,
{
    type Rejection = Response;

    async fn from_request_parts(parts: &mut Parts, state: &S) -> Result<Self, Self::Rejection> {
        let state = Arc::<AppState>::from_ref(state);
        let token = extract_session_cookie(&parts.headers);

        if let Some(token) = token {
            if is_valid_session(&state, &token).await {
                return Ok(AuthSession);
            }
        }

        Err(Redirect::to("/login").into_response())
    }
}

async fn is_valid_session(state: &AppState, token: &str) -> bool {
    match crate::db::get_session_expiry(&state.db, token) {
        Ok(Some(expires_at)) => {
            use std::time::{SystemTime, UNIX_EPOCH};
            let now_secs = SystemTime::now()
                .duration_since(UNIX_EPOCH)
                .unwrap()
                .as_secs();
            let now_str = {
                let days = now_secs / 86400;
                let h = (now_secs / 3600) % 24;
                let m = (now_secs / 60) % 60;
                let s = now_secs % 60;
                format_unix_to_datetime(days, h, m, s)
            };
            expires_at > now_str
        }
        _ => false,
    }
}
