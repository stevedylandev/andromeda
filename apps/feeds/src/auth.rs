use axum::{
    extract::{FromRef, FromRequestParts},
    http::request::Parts,
    response::{IntoResponse, Redirect, Response},
};
use std::collections::HashMap;
use std::sync::{Arc, Mutex};
use std::time::{Duration, Instant};

use crate::AppState;

pub use andromeda_auth::{
    build_session_cookie, clear_session_cookie, extract_session_cookie, generate_session_token,
    verify_password,
};

pub type SessionStore = Arc<Mutex<HashMap<String, Instant>>>;

const SESSION_TTL: Duration = Duration::from_secs(7 * 24 * 60 * 60); // 7 days

pub fn new_session_store() -> SessionStore {
    Arc::new(Mutex::new(HashMap::new()))
}

pub fn create_session(store: &SessionStore, token: &str) {
    if let Ok(mut sessions) = store.lock() {
        sessions.insert(token.to_string(), Instant::now());
    }
}

pub fn is_valid_session(store: &SessionStore, token: &str) -> bool {
    if let Ok(mut sessions) = store.lock() {
        if let Some(created) = sessions.get(token) {
            if created.elapsed() < SESSION_TTL {
                return true;
            }
            sessions.remove(token);
        }
    }
    false
}

pub fn delete_session(store: &SessionStore, token: &str) {
    if let Ok(mut sessions) = store.lock() {
        sessions.remove(token);
    }
}

/// Axum extractor — guards routes behind login. Redirects to /admin/login if invalid.
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
            if is_valid_session(&state.sessions, &token) {
                return Ok(AuthSession);
            }
        }
        Err(Redirect::to("/admin/login").into_response())
    }
}
