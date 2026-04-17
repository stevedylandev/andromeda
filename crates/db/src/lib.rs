use rusqlite::Connection;
use std::fmt;
use std::sync::{Arc, Mutex};

pub type Db = Arc<Mutex<Connection>>;

pub trait HasDb {
    fn db(&self) -> &Db;
}

#[derive(Debug)]
pub enum DbError {
    Sqlite(rusqlite::Error),
    LockPoisoned,
}

impl fmt::Display for DbError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            DbError::Sqlite(e) => write!(f, "Database error: {}", e),
            DbError::LockPoisoned => write!(f, "Database lock poisoned"),
        }
    }
}

impl std::error::Error for DbError {}

impl From<rusqlite::Error> for DbError {
    fn from(e: rusqlite::Error) -> Self {
        DbError::Sqlite(e)
    }
}

#[cfg(feature = "axum")]
impl axum::response::IntoResponse for DbError {
    fn into_response(self) -> axum::response::Response {
        tracing::error!("{}", self);
        (
            axum::http::StatusCode::INTERNAL_SERVER_ERROR,
            "Server error",
        )
            .into_response()
    }
}

#[cfg(feature = "session")]
pub mod session;
