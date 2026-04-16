use crate::db::{self, Db, Note, NoteInput};
use reqwest::StatusCode;
use reqwest::blocking::{Client, RequestBuilder, Response};
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

fn net<E: fmt::Display>(e: E) -> BackendError {
    BackendError::Network(e.to_string())
}

fn with_key(req: RequestBuilder, key: &Option<String>) -> RequestBuilder {
    match key {
        Some(k) => req.header("x-api-key", k),
        None => req,
    }
}

fn send_request(req: RequestBuilder) -> Result<Response, BackendError> {
    let resp = req.send().map_err(net)?;
    match resp.status().as_u16() {
        401 => Err(BackendError::Unauthorized("Invalid API key".into())),
        403 => Err(BackendError::Unauthorized(
            "No API key configured on server".into(),
        )),
        _ => Ok(resp),
    }
}

fn unexpected(status: StatusCode) -> BackendError {
    BackendError::Network(format!("HTTP {}", status))
}

pub enum Backend {
    Local {
        db: Db,
    },
    Remote {
        base_url: String,
        api_key: Option<String>,
        client: Client,
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
            client: Client::new(),
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
                let req = with_key(client.get(format!("{base_url}/api/notes")), api_key);
                let resp = send_request(req)?;
                match resp.status().as_u16() {
                    200 => resp.json::<Vec<Note>>().map_err(net),
                    _ => Err(unexpected(resp.status())),
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
                let body = NoteInput {
                    title: title.to_string(),
                    content: content.to_string(),
                };
                let req = with_key(
                    client.post(format!("{base_url}/api/notes")).json(&body),
                    api_key,
                );
                let resp = send_request(req)?;
                match resp.status().as_u16() {
                    201 => resp.json::<Note>().map_err(net),
                    _ => Err(unexpected(resp.status())),
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
                let body = NoteInput {
                    title: title.to_string(),
                    content: content.to_string(),
                };
                let req = with_key(
                    client
                        .put(format!("{base_url}/api/notes/{short_id}"))
                        .json(&body),
                    api_key,
                );
                let resp = send_request(req)?;
                match resp.status().as_u16() {
                    200 => resp.json::<Note>().map(Some).map_err(net),
                    404 => Ok(None),
                    _ => Err(unexpected(resp.status())),
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
                let req = with_key(
                    client.delete(format!("{base_url}/api/notes/{short_id}")),
                    api_key,
                );
                let resp = send_request(req)?;
                match resp.status().as_u16() {
                    200 | 204 => Ok(true),
                    404 => Ok(false),
                    _ => Err(unexpected(resp.status())),
                }
            }
        }
    }
}
