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

/// Return an ISO datetime string 7 days from now.
pub fn session_expiry_at() -> String {
    andromeda_auth::datetime::expiry_datetime_string(7 * 24 * 3600)
}

pub fn extract_session_token(headers: &axum::http::HeaderMap) -> Option<String> {
    extract_session_cookie(headers)
}

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
        Ok(Some(expires_at)) => expires_at > andromeda_auth::datetime::now_datetime_string(),
        _ => false,
    }
}
