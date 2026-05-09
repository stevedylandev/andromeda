use rand::Rng;
use serde::Deserialize;
use std::time::Duration;

use crate::db::{self, DailyArtwork, Db};

const SEARCH_URL: &str = "https://api.artic.edu/api/v1/artworks/search";
const FIELDS: &str = "id,title,artist_display,artist_title,date_display,medium_display,dimensions,place_of_origin,credit_line,description,short_description,image_id";

pub fn build_client() -> reqwest::Client {
    reqwest::Client::builder()
        .timeout(Duration::from_secs(20))
        .user_agent("andromeda-easel/0.1 (+https://github.com/stevedylandev/andromeda)")
        .build()
        .expect("Failed to build HTTP client")
}

#[derive(Debug, Deserialize)]
pub struct RawArtwork {
    pub id: i64,
    pub title: Option<String>,
    pub artist_display: Option<String>,
    pub artist_title: Option<String>,
    pub date_display: Option<String>,
    pub medium_display: Option<String>,
    pub dimensions: Option<String>,
    pub place_of_origin: Option<String>,
    pub credit_line: Option<String>,
    pub description: Option<String>,
    pub short_description: Option<String>,
    pub image_id: Option<String>,
}

#[derive(Debug, Deserialize)]
struct Pagination {
    total: u64,
}

#[derive(Debug, Deserialize)]
struct SearchResponse<T> {
    pagination: Pagination,
    data: Vec<T>,
}

#[derive(Debug, Deserialize)]
struct IdOnly {
    #[allow(dead_code)]
    id: i64,
}

fn build_params(classifications: &[String]) -> String {
    let terms: Vec<serde_json::Value> = classifications
        .iter()
        .map(|c| serde_json::Value::String(c.to_lowercase()))
        .collect();
    let body = serde_json::json!({
        "query": {
            "bool": {
                "must": [
                    { "term": { "is_public_domain": true } },
                    { "terms": { "classification_title.keyword": terms } },
                    { "exists": { "field": "image_id" } }
                ]
            }
        }
    });
    body.to_string()
}

pub async fn total_matching(
    client: &reqwest::Client,
    classifications: &[String],
) -> Result<u64, String> {
    let params = build_params(classifications);
    let url = format!(
        "{SEARCH_URL}?params={}&limit=1&fields=id",
        urlencoding::encode(&params)
    );
    let resp = client
        .get(&url)
        .send()
        .await
        .map_err(|e| format!("count fetch failed: {e}"))?;
    if !resp.status().is_success() {
        return Err(format!("count returned status {}", resp.status()));
    }
    let body: SearchResponse<IdOnly> = resp
        .json()
        .await
        .map_err(|e| format!("count parse failed: {e}"))?;
    Ok(body.pagination.total)
}

pub async fn fetch_artwork_at(
    client: &reqwest::Client,
    classifications: &[String],
    page: u64,
) -> Result<Option<RawArtwork>, String> {
    let params = build_params(classifications);
    let url = format!(
        "{SEARCH_URL}?params={}&limit=1&page={page}&fields={FIELDS}",
        urlencoding::encode(&params)
    );
    let resp = client
        .get(&url)
        .send()
        .await
        .map_err(|e| format!("artwork fetch failed: {e}"))?;
    if !resp.status().is_success() {
        return Err(format!("artwork returned status {}", resp.status()));
    }
    let mut body: SearchResponse<RawArtwork> = resp
        .json()
        .await
        .map_err(|e| format!("artwork parse failed: {e}"))?;
    Ok(body.data.pop())
}

pub async fn pick_unique(
    client: &reqwest::Client,
    db: &Db,
    classifications: &[String],
    max_retries: u32,
) -> Result<RawArtwork, String> {
    let total = total_matching(client, classifications).await?;
    if total == 0 {
        return Err("AIC search returned zero matches for given classifications".to_string());
    }

    for attempt in 0..=max_retries {
        let page = {
            let mut rng = rand::thread_rng();
            rng.gen_range(1..=total)
        };
        let art = match fetch_artwork_at(client, classifications, page).await? {
            Some(a) => a,
            None => continue,
        };
        if art.image_id.is_none() || art.image_id.as_deref() == Some("") {
            tracing::warn!("artwork {} has no image_id, retrying", art.id);
            continue;
        }
        match db::artwork_id_exists(db, art.id) {
            Ok(true) => {
                tracing::info!(
                    "duplicate artwork {} on attempt {}, retrying",
                    art.id,
                    attempt + 1
                );
                continue;
            }
            Ok(false) => return Ok(art),
            Err(e) => return Err(format!("dedup check failed: {e}")),
        }
    }
    Err(format!(
        "failed to pick a non-duplicate artwork after {} retries",
        max_retries + 1
    ))
}

pub fn raw_to_daily(raw: RawArtwork, date: String, fetched_at: String) -> Option<DailyArtwork> {
    let image_id = raw.image_id?;
    if image_id.is_empty() {
        return None;
    }
    Some(DailyArtwork {
        date,
        artwork_id: raw.id,
        title: raw.title.unwrap_or_else(|| "Untitled".to_string()),
        artist_display: raw.artist_display,
        artist_title: raw.artist_title,
        date_display: raw.date_display,
        medium_display: raw.medium_display,
        dimensions: raw.dimensions,
        place_of_origin: raw.place_of_origin,
        credit_line: raw.credit_line,
        description: raw.description,
        short_description: raw.short_description,
        image_id,
        fetched_at,
    })
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn build_params_lowercases_classifications() {
        let p = build_params(&["Painting".to_string(), "DRAWING".to_string()]);
        assert!(p.contains("\"painting\""));
        assert!(p.contains("\"drawing\""));
        assert!(p.contains("is_public_domain"));
        assert!(p.contains("image_id"));
    }
}
