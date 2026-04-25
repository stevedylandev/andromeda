use serde::{Deserialize, Serialize};
use std::time::Duration;

#[derive(Debug, Clone, Serialize)]
pub struct SearchHit {
    pub google_id: String,
    pub title: String,
    pub authors: String,
    pub isbn: Option<String>,
    pub cover_url: Option<String>,
}

#[derive(Deserialize)]
struct VolumesResponse {
    #[serde(default)]
    items: Vec<Volume>,
}

#[derive(Deserialize)]
struct Volume {
    id: String,
    #[serde(rename = "volumeInfo")]
    volume_info: VolumeInfo,
}

#[derive(Deserialize)]
struct VolumeInfo {
    title: Option<String>,
    #[serde(default)]
    authors: Vec<String>,
    #[serde(default, rename = "industryIdentifiers")]
    identifiers: Vec<Identifier>,
    #[serde(rename = "imageLinks")]
    image_links: Option<ImageLinks>,
}

#[derive(Deserialize)]
struct Identifier {
    #[serde(rename = "type")]
    kind: String,
    identifier: String,
}

#[derive(Deserialize)]
struct ImageLinks {
    thumbnail: Option<String>,
    #[serde(rename = "smallThumbnail")]
    small_thumbnail: Option<String>,
}

pub async fn search(query: &str, api_key: Option<&str>) -> Result<Vec<SearchHit>, String> {
    let trimmed = query.trim();
    if trimmed.is_empty() {
        return Ok(Vec::new());
    }
    let normalized: String = trimmed
        .chars()
        .filter(|c| !c.is_whitespace() && *c != '-')
        .collect();
    let is_isbn = matches!(normalized.len(), 10 | 13)
        && normalized
            .chars()
            .all(|c| c.is_ascii_digit() || c == 'X' || c == 'x');
    let query_str = if is_isbn {
        format!("isbn:{}", normalized.to_uppercase())
    } else {
        trimmed.to_string()
    };
    let q = urlencoding::encode(&query_str);
    let mut url = format!(
        "https://www.googleapis.com/books/v1/volumes?q={q}&maxResults=10&printType=books"
    );
    if let Some(key) = api_key {
        url.push_str(&format!("&key={}", urlencoding::encode(key)));
    }

    let client = reqwest::Client::builder()
        .timeout(Duration::from_secs(8))
        .user_agent("andromeda-library/0.1")
        .build()
        .map_err(|e| format!("client build: {e}"))?;

    let resp = client
        .get(&url)
        .send()
        .await
        .map_err(|e| format!("request: {e}"))?;

    if !resp.status().is_success() {
        return Err(format!("google books status {}", resp.status()));
    }

    let data: VolumesResponse = resp
        .json()
        .await
        .map_err(|e| format!("parse: {e}"))?;

    Ok(data
        .items
        .into_iter()
        .map(|v| {
            let info = v.volume_info;
            let isbn = pick_isbn(&info.identifiers);
            let cover_url = info
                .image_links
                .as_ref()
                .and_then(|l| l.thumbnail.clone().or_else(|| l.small_thumbnail.clone()))
                .map(|u| u.replacen("http://", "https://", 1));
            SearchHit {
                google_id: v.id,
                title: info.title.unwrap_or_else(|| "Untitled".to_string()),
                authors: info.authors.join(", "),
                isbn,
                cover_url,
            }
        })
        .collect())
}

fn pick_isbn(ids: &[Identifier]) -> Option<String> {
    ids.iter()
        .find(|i| i.kind == "ISBN_13")
        .or_else(|| ids.iter().find(|i| i.kind == "ISBN_10"))
        .map(|i| i.identifier.clone())
}
