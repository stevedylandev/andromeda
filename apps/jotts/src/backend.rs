use crate::db::{self, Db, Note};
use std::fmt;

#[derive(Debug)]
pub enum BackendError {
    #[allow(dead_code)]
    NotFound,
    Unauthorized(String),
    Network(String),
    Database(String),
}

impl fmt::Display for BackendError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            BackendError::NotFound => write!(f, "Not found"),
            BackendError::Unauthorized(msg) => write!(f, "Unauthorized: {}", msg),
            BackendError::Network(msg) => write!(f, "Network error: {}", msg),
            BackendError::Database(msg) => write!(f, "Database error: {}", msg),
        }
    }
}

impl std::error::Error for BackendError {}

impl From<db::DbError> for BackendError {
    fn from(e: db::DbError) -> Self {
        BackendError::Database(e.to_string())
    }
}

pub enum Backend {
    Local {
        db: Db,
    },
    Remote {
        base_url: String,
        api_key: Option<String>,
        client: reqwest::blocking::Client,
    },
}

impl Backend {
    pub fn local() -> Self {
        Backend::Local { db: db::init_db() }
    }

    pub fn remote(base_url: String, api_key: Option<String>) -> Self {
        Backend::Remote {
            base_url,
            api_key,
            client: reqwest::blocking::Client::new(),
        }
    }

    pub fn list_notes(&self) -> Result<Vec<Note>, BackendError> {
        match self {
            Backend::Local { db } => Ok(db::get_all_notes(db)?),
            Backend::Remote {
                base_url,
                api_key,
                client,
            } => {
                let mut req = client.get(format!("{}/api/notes", base_url));
                if let Some(key) = api_key {
                    req = req.header("x-api-key", key);
                }
                let resp = req
                    .send()
                    .map_err(|e| BackendError::Network(e.to_string()))?;
                match resp.status().as_u16() {
                    200 => resp
                        .json::<Vec<Note>>()
                        .map_err(|e| BackendError::Network(e.to_string())),
                    401 => Err(BackendError::Unauthorized("Invalid API key".into())),
                    403 => Err(BackendError::Unauthorized(
                        "No API key configured on server".into(),
                    )),
                    _ => Err(BackendError::Network(format!("HTTP {}", resp.status()))),
                }
            }
        }
    }

    pub fn create_note(&self, title: &str, content: &str) -> Result<Note, BackendError> {
        match self {
            Backend::Local { db } => Ok(db::create_note(db, title, content)?),
            Backend::Remote {
                base_url,
                api_key,
                client,
            } => {
                let mut req = client
                    .post(format!("{}/api/notes", base_url))
                    .json(&serde_json::json!({"title": title, "content": content}));
                if let Some(key) = api_key {
                    req = req.header("x-api-key", key);
                }
                let resp = req
                    .send()
                    .map_err(|e| BackendError::Network(e.to_string()))?;
                match resp.status().as_u16() {
                    201 => resp
                        .json::<Note>()
                        .map_err(|e| BackendError::Network(e.to_string())),
                    401 => Err(BackendError::Unauthorized("Invalid API key".into())),
                    403 => Err(BackendError::Unauthorized(
                        "No API key configured on server".into(),
                    )),
                    _ => Err(BackendError::Network(format!("HTTP {}", resp.status()))),
                }
            }
        }
    }

    pub fn update_note(
        &self,
        short_id: &str,
        title: &str,
        content: &str,
    ) -> Result<Option<Note>, BackendError> {
        match self {
            Backend::Local { db } => Ok(db::update_note_by_short_id(db, short_id, title, content)?),
            Backend::Remote {
                base_url,
                api_key,
                client,
            } => {
                let mut req = client
                    .put(format!("{}/api/notes/{}", base_url, short_id))
                    .json(&serde_json::json!({"title": title, "content": content}));
                if let Some(key) = api_key {
                    req = req.header("x-api-key", key);
                }
                let resp = req
                    .send()
                    .map_err(|e| BackendError::Network(e.to_string()))?;
                match resp.status().as_u16() {
                    200 => resp
                        .json::<Note>()
                        .map(Some)
                        .map_err(|e| BackendError::Network(e.to_string())),
                    401 => Err(BackendError::Unauthorized("Invalid API key".into())),
                    403 => Err(BackendError::Unauthorized(
                        "No API key configured on server".into(),
                    )),
                    404 => Ok(None),
                    _ => Err(BackendError::Network(format!("HTTP {}", resp.status()))),
                }
            }
        }
    }

    pub fn delete_note(&self, short_id: &str) -> Result<bool, BackendError> {
        match self {
            Backend::Local { db } => Ok(db::delete_note_by_short_id(db, short_id)?),
            Backend::Remote {
                base_url,
                api_key,
                client,
            } => {
                let mut req = client.delete(format!("{}/api/notes/{}", base_url, short_id));
                if let Some(key) = api_key {
                    req = req.header("x-api-key", key);
                }
                let resp = req
                    .send()
                    .map_err(|e| BackendError::Network(e.to_string()))?;
                match resp.status().as_u16() {
                    200 | 204 => Ok(true),
                    401 => Err(BackendError::Unauthorized("Invalid API key".into())),
                    403 => Err(BackendError::Unauthorized(
                        "No API key configured on server".into(),
                    )),
                    404 => Ok(false),
                    _ => Err(BackendError::Network(format!("HTTP {}", resp.status()))),
                }
            }
        }
    }
}

