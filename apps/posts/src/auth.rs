use axum::{
    extract::FromRequestParts,
    http::request::Parts,
    response::{IntoResponse, Redirect, Response},
};
use std::sync::Arc;

use crate::db;
use crate::server::AppState;

pub use andromeda_auth::{
    build_session_cookie, clear_session_cookie, generate_session_token, verify_password,
};

pub struct AuthSession;

impl FromRequestParts<Arc<AppState>> for AuthSession {
    type Rejection = Response;

    async fn from_request_parts(
        parts: &mut Parts,
        state: &Arc<AppState>,
    ) -> Result<Self, Self::Rejection> {
        let token = andromeda_auth::extract_session_cookie(&parts.headers);
        if let Some(token) = token {
            if is_valid_session(state, &token) {
                return Ok(AuthSession);
            }
        }
        Err(Redirect::to("/admin/login").into_response())
    }
}

fn is_valid_session(state: &AppState, token: &str) -> bool {
    match db::get_session_expiry(&state.db, token) {
        Ok(Some(expires_at)) => expires_at > andromeda_auth::datetime::now_datetime_string(),
        _ => false,
    }
}
