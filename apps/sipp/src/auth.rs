use axum::{
    extract::FromRequestParts,
    http::request::Parts,
    response::{IntoResponse, Redirect, Response},
};

use crate::db;
use crate::server::AppState;

pub use andromeda_auth::{
    build_session_cookie, clear_session_cookie, generate_session_token, verify_api_key,
};

pub struct AuthSession;

impl FromRequestParts<AppState> for AuthSession {
    type Rejection = Response;

    async fn from_request_parts(
        parts: &mut Parts,
        state: &AppState,
    ) -> Result<Self, Self::Rejection> {
        if let Some(token) = andromeda_auth::extract_session_cookie(&parts.headers) {
            if is_valid_session(state, &token) {
                return Ok(AuthSession);
            }
        }
        let path = parts
            .uri
            .path_and_query()
            .map(|pq| pq.as_str())
            .unwrap_or(parts.uri.path());
        let login_url = format!("/admin/login?next={}", urlencoding(path));
        Err(Redirect::to(&login_url).into_response())
    }
}

pub fn is_valid_session(state: &AppState, token: &str) -> bool {
    match db::get_session_expiry(&state.db, token) {
        Ok(Some(expires_at)) => expires_at > andromeda_auth::datetime::now_datetime_string(),
        _ => false,
    }
}

pub fn urlencoding(s: &str) -> String {
    let mut out = String::with_capacity(s.len());
    for b in s.bytes() {
        match b {
            b'A'..=b'Z' | b'a'..=b'z' | b'0'..=b'9' | b'-' | b'_' | b'.' | b'~' | b'/' => {
                out.push(b as char);
            }
            _ => {
                out.push_str(&format!("%{:02X}", b));
            }
        }
    }
    out
}
