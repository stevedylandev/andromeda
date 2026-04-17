use crate::db::{self, Db, Snippet, SnippetInput};
use reqwest::StatusCode;
use reqwest::blocking::{Client, RequestBuilder, Response};
use std::fmt;

#[derive(Debug)]
pub enum BackendError {
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
    pub fn local() -> Result<Self, BackendError> {
        Ok(Backend::Local { db: db::init_db()? })
    }

    pub fn remote(base_url: String, api_key: Option<String>) -> Self {
        Backend::Remote {
            base_url,
            api_key,
            client: Client::new(),
        }
    }

    pub fn list_snippets(&self) -> Result<Vec<Snippet>, BackendError> {
        match self {
            Backend::Local { db } => Ok(db::get_all_snippets(db)?),
            Backend::Remote {
                base_url,
                api_key,
                client,
            } => {
                let req = with_key(client.get(format!("{base_url}/api/snippets")), api_key);
                let resp = send_request(req)?;
                match resp.status().as_u16() {
                    200 => resp.json::<Vec<Snippet>>().map_err(net),
                    _ => Err(unexpected(resp.status())),
                }
            }
        }
    }

    pub fn create_snippet(&self, name: &str, content: &str) -> Result<Snippet, BackendError> {
        match self {
            Backend::Local { db } => Ok(db::create_snippet(db, name, content)?),
            Backend::Remote {
                base_url,
                api_key,
                client,
            } => {
                let body = SnippetInput {
                    name: name.to_string(),
                    content: content.to_string(),
                };
                let req = with_key(
                    client.post(format!("{base_url}/api/snippets")).json(&body),
                    api_key,
                );
                let resp = send_request(req)?;
                match resp.status().as_u16() {
                    201 => resp.json::<Snippet>().map_err(net),
                    _ => Err(unexpected(resp.status())),
                }
            }
        }
    }

    pub fn update_snippet(
        &self,
        short_id: &str,
        name: &str,
        content: &str,
    ) -> Result<Option<Snippet>, BackendError> {
        match self {
            Backend::Local { db } => Ok(db::update_snippet_by_short_id(db, short_id, name, content)?),
            Backend::Remote {
                base_url,
                api_key,
                client,
            } => {
                let body = SnippetInput {
                    name: name.to_string(),
                    content: content.to_string(),
                };
                let req = with_key(
                    client
                        .put(format!("{base_url}/api/snippets/{short_id}"))
                        .json(&body),
                    api_key,
                );
                let resp = send_request(req)?;
                match resp.status().as_u16() {
                    200 => resp.json::<Snippet>().map(Some).map_err(net),
                    404 => Ok(None),
                    _ => Err(unexpected(resp.status())),
                }
            }
        }
    }

    pub fn delete_snippet(&self, short_id: &str) -> Result<bool, BackendError> {
        match self {
            Backend::Local { db } => Ok(db::delete_snippet_by_short_id(db, short_id)?),
            Backend::Remote {
                base_url,
                api_key,
                client,
            } => {
                let req = with_key(
                    client.delete(format!("{base_url}/api/snippets/{short_id}")),
                    api_key,
                );
                let resp = send_request(req)?;
                match resp.status().as_u16() {
                    200 => Ok(true),
                    404 => Ok(false),
                    _ => Err(unexpected(resp.status())),
                }
            }
        }
    }
}
