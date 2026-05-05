use reqwest::blocking::{Client, RequestBuilder, Response};
use serde::{Deserialize, Serialize};
use std::fmt;

#[derive(Debug)]
pub enum BackendError {
    Unauthorized(String),
    Network(String),
    NotFound,
}

impl fmt::Display for BackendError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            BackendError::Unauthorized(m) => write!(f, "Unauthorized: {m}"),
            BackendError::Network(m) => write!(f, "Network error: {m}"),
            BackendError::NotFound => write!(f, "Not found"),
        }
    }
}

impl std::error::Error for BackendError {}

fn net<E: fmt::Display>(e: E) -> BackendError {
    BackendError::Network(e.to_string())
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ListItem {
    pub title: String,
    pub link: String,
    #[serde(default)]
    pub author: Option<String>,
    #[serde(default)]
    pub feed_title: Option<String>,
    pub published_at: i64,
}

#[derive(Debug, Clone, Deserialize)]
struct ItemsResponse {
    items: Vec<serde_json::Value>,
}

#[derive(Debug, Clone, Deserialize)]
pub struct DiscoverResponse {
    pub feeds: Vec<String>,
}

#[derive(Debug, Clone, Deserialize, Default)]
pub struct Category {
    pub id: i64,
    pub name: String,
}

pub struct Backend {
    base_url: String,
    api_key: Option<String>,
    client: Client,
}

impl Backend {
    pub fn new(base_url: String, api_key: Option<String>) -> Self {
        Self {
            base_url,
            api_key,
            client: Client::builder()
                .timeout(std::time::Duration::from_secs(20))
                .build()
                .expect("build client"),
        }
    }

    pub fn base_url(&self) -> &str {
        &self.base_url
    }

    pub fn has_key(&self) -> bool {
        self.api_key.is_some()
    }

    fn auth(&self, req: RequestBuilder) -> RequestBuilder {
        match &self.api_key {
            Some(k) => req.bearer_auth(k),
            None => req,
        }
    }

    fn send(&self, req: RequestBuilder) -> Result<Response, BackendError> {
        let resp = req.send().map_err(net)?;
        match resp.status().as_u16() {
            401 | 403 => Err(BackendError::Unauthorized("invalid or missing API key".into())),
            404 => Err(BackendError::NotFound),
            _ => Ok(resp),
        }
    }

    /// List the recent items from the server's aggregated feed.
    pub fn list_items(&self, limit: i64) -> Result<Vec<ListItem>, BackendError> {
        let url = format!("{}/api/items?limit={limit}", self.base_url.trim_end_matches('/'));
        let resp = self.send(self.client.get(url))?;
        if !resp.status().is_success() {
            return Err(BackendError::Network(format!("HTTP {}", resp.status())));
        }
        let parsed: ItemsResponse = resp.json().map_err(net)?;
        Ok(parsed.items.into_iter().filter_map(parse_item).collect())
    }

    /// Preview a single feed URL on demand. No DB write, no auth.
    pub fn preview(&self, url: &str) -> Result<Vec<ListItem>, BackendError> {
        let endpoint = format!(
            "{}/api/preview?url={}",
            self.base_url.trim_end_matches('/'),
            urlencoding::encode(url),
        );
        let resp = self.send(self.client.get(endpoint))?;
        if !resp.status().is_success() {
            return Err(BackendError::Network(format!("HTTP {}", resp.status())));
        }
        let parsed: ItemsResponse = resp.json().map_err(net)?;
        Ok(parsed.items.into_iter().filter_map(parse_item).collect())
    }

    pub fn discover(&self, base_url: &str) -> Result<Vec<String>, BackendError> {
        let endpoint = format!("{}/api/discover", self.base_url.trim_end_matches('/'));
        let body = serde_json::json!({ "base_url": base_url });
        let resp = self.send(self.auth(self.client.post(endpoint).json(&body)))?;
        let status = resp.status();
        if !status.is_success() {
            let msg = resp.text().unwrap_or_default();
            return Err(BackendError::Network(format!("HTTP {status}: {msg}")));
        }
        let parsed: DiscoverResponse = resp.json().map_err(net)?;
        Ok(parsed.feeds)
    }

    pub fn add_subscription(
        &self,
        feed_url: &str,
        category_name: Option<&str>,
    ) -> Result<(), BackendError> {
        let endpoint = format!("{}/api/subscriptions", self.base_url.trim_end_matches('/'));
        let mut body = serde_json::json!({ "feed_url": feed_url });
        if let Some(name) = category_name.filter(|s| !s.trim().is_empty()) {
            body["category_name"] = serde_json::Value::String(name.to_string());
        }
        let resp = self.send(self.auth(self.client.post(endpoint).json(&body)))?;
        let status = resp.status();
        if status.is_success() {
            return Ok(());
        }
        if status.as_u16() == 409 {
            return Err(BackendError::Network("already subscribed".into()));
        }
        let msg = resp.text().unwrap_or_default();
        Err(BackendError::Network(format!("HTTP {status}: {msg}")))
    }

    pub fn list_categories(&self) -> Result<Vec<Category>, BackendError> {
        let endpoint = format!("{}/api/categories", self.base_url.trim_end_matches('/'));
        let resp = self.send(self.auth(self.client.get(endpoint)))?;
        if !resp.status().is_success() {
            return Err(BackendError::Network(format!("HTTP {}", resp.status())));
        }
        #[derive(Deserialize)]
        struct CatsResp {
            categories: Vec<Category>,
        }
        let parsed: CatsResp = resp.json().map_err(net)?;
        Ok(parsed.categories)
    }
}

fn parse_item(v: serde_json::Value) -> Option<ListItem> {
    // Server-side returns either ItemWithFeed (DB items) or FeedItem (preview).
    // ItemWithFeed: title, link, author, published_at, feed_title.
    // FeedItem: title, link, author (String), published.
    let title = v.get("title")?.as_str()?.to_string();
    let link = v.get("link")?.as_str()?.to_string();
    let author = v
        .get("author")
        .and_then(|a| a.as_str())
        .map(|s| s.to_string());
    let feed_title = v
        .get("feed_title")
        .and_then(|a| a.as_str())
        .map(|s| s.to_string());
    let published_at = v
        .get("published_at")
        .or_else(|| v.get("published"))
        .and_then(|p| p.as_i64())
        .unwrap_or(0);
    Some(ListItem {
        title,
        link,
        author,
        feed_title,
        published_at,
    })
}
