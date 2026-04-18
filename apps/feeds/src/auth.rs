use axum::{
    extract::{FromRef, FromRequestParts},
    http::{request::Parts, StatusCode},
    response::{IntoResponse, Redirect, Response},
};
use chrono::{Duration, Utc};
use std::sync::Arc;

use crate::AppState;
use andromeda_db::session;

pub use andromeda_auth::{
    build_session_cookie, clear_session_cookie, extract_session_cookie, generate_session_token,
    verify_api_key, verify_password,
};

const SESSION_DAYS: i64 = 7;

/// Create a session row with 7-day expiry.
pub fn create_session(db: &andromeda_db::Db, token: &str) -> Result<(), andromeda_db::DbError> {
    let expires = (Utc::now() + Duration::days(SESSION_DAYS))
        .format("%Y-%m-%d %H:%M:%S")
        .to_string();
    session::insert_session(db, token, &expires)
}

pub fn is_valid_session(db: &andromeda_db::Db, token: &str) -> bool {
    match session::get_session_expiry(db, token) {
        Ok(Some(expires_at)) => {
            chrono::NaiveDateTime::parse_from_str(&expires_at, "%Y-%m-%d %H:%M:%S")
                .map(|exp| exp > Utc::now().naive_utc())
                .unwrap_or(false)
        }
        _ => false,
    }
}

pub fn delete_session(db: &andromeda_db::Db, token: &str) {
    let _ = session::delete_session(db, token);
}

/// Guards browser admin routes. Redirects to login on failure.
pub struct AuthSession;

impl<S> FromRequestParts<S> for AuthSession
where
    S: Send + Sync,
    Arc<AppState>: FromRef<S>,
{
    type Rejection = Response;

    async fn from_request_parts(parts: &mut Parts, state: &S) -> Result<Self, Self::Rejection> {
        let state = Arc::<AppState>::from_ref(state);
        if let Some(token) = extract_session_cookie(&parts.headers) {
            if is_valid_session(&state.db, &token) {
                return Ok(AuthSession);
            }
        }
        Err(Redirect::to("/admin/login").into_response())
    }
}

/// Guards JSON API routes. Accepts `Authorization: Bearer <API_KEY>` OR a valid session cookie.
/// Returns 401 JSON on failure (doesn't redirect).
pub struct ApiAuth;

impl<S> FromRequestParts<S> for ApiAuth
where
    S: Send + Sync,
    Arc<AppState>: FromRef<S>,
{
    type Rejection = Response;

    async fn from_request_parts(parts: &mut Parts, state: &S) -> Result<Self, Self::Rejection> {
        let state = Arc::<AppState>::from_ref(state);

        if let Some(expected_key) = state.api_key.as_deref() {
            if let Some(header) = parts.headers.get(axum::http::header::AUTHORIZATION) {
                if let Ok(s) = header.to_str() {
                    if let Some(token) = s.strip_prefix("Bearer ").or_else(|| s.strip_prefix("bearer ")) {
                        if verify_api_key(token.trim(), expected_key) {
                            return Ok(ApiAuth);
                        }
                    }
                }
            }
        }

        if let Some(token) = extract_session_cookie(&parts.headers) {
            if is_valid_session(&state.db, &token) {
                return Ok(ApiAuth);
            }
        }

        Err((
            StatusCode::UNAUTHORIZED,
            axum::Json(serde_json::json!({ "error": "unauthorized" })),
        )
            .into_response())
    }
}
