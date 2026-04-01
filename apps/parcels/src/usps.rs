use reqwest::Client;
use serde::{Deserialize, Serialize};
use std::time::{Duration, Instant};

// ── Token Cache ────────────────────────────────────────────────────────────

pub struct CachedToken {
    pub token: String,
    pub expires_at: Instant,
}

// ── USPS OAuth2 Response ───────────────────────────────────────────────────

#[derive(Deserialize)]
struct TokenResponse {
    access_token: String,
    expires_in: u64,
}

// ── Tracking Request/Response Types ────────────────────────────────────────

#[derive(Serialize)]
struct TrackingRequestBody {
    #[serde(rename = "trackingNumber")]
    tracking_number: String,
}

#[derive(Debug, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct TrackingDetail {
    pub tracking_number: Option<String>,
    pub status: Option<String>,
    pub status_category: Option<String>,
    pub status_summary: Option<String>,
    pub mail_class: Option<String>,
    pub delivery_date_expectation: Option<DeliveryDateExpectation>,
    #[serde(default)]
    pub tracking_events: Vec<TrackingEvent>,
}

#[derive(Debug, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct DeliveryDateExpectation {
    pub expected_delivery_date: Option<String>,
}

#[derive(Debug, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct TrackingEvent {
    pub event_timestamp: Option<String>,
    pub event_type: Option<String>,
    pub event_city: Option<String>,
    pub event_state: Option<String>,
    #[serde(rename = "eventZIPCode")]
    pub event_zip_code: Option<String>,
    pub event_code: Option<String>,
}

// ── Multi-status (207) response ────────────────────────────────────────────

#[derive(Debug, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct FailureResponse {
    #[serde(default)]
    pub status_code: String,
    pub error: Option<ErrorObject>,
}

#[derive(Debug, Deserialize)]
pub struct ErrorObject {
    pub message: Option<String>,
}

// ── Errors ─────────────────────────────────────────────────────────────────

#[derive(Debug)]
pub enum UspsError {
    NotFoundOrInvalid(String),
    BadRequest,
    Unauthorized,
    RateLimit,
    ServiceUnavailable,
    Timeout,
    Other(String),
}

impl std::fmt::Display for UspsError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            UspsError::NotFoundOrInvalid(msg) => write!(f, "{}", msg),
            UspsError::BadRequest => write!(f, "Invalid request sent to USPS API. Check the tracking number format."),
            UspsError::Unauthorized => write!(f, "USPS credentials are invalid. Check USPS_CLIENT_ID and USPS_CLIENT_SECRET."),
            UspsError::RateLimit => write!(f, "USPS rate limit hit. Try again shortly."),
            UspsError::ServiceUnavailable => write!(f, "USPS service unavailable. Try again later."),
            UspsError::Timeout => write!(f, "USPS request timed out."),
            UspsError::Other(msg) => write!(f, "{}", msg),
        }
    }
}

impl std::error::Error for UspsError {}

// ── Token Fetch ────────────────────────────────────────────────────────────

pub async fn fetch_token(
    client: &Client,
    client_id: &str,
    client_secret: &str,
) -> Result<CachedToken, UspsError> {
    let params = [
        ("grant_type", "client_credentials"),
        ("client_id", client_id),
        ("client_secret", client_secret),
        ("scope", "tracking"),
    ];

    let resp = client
        .post("https://apis.usps.com/oauth2/v3/token")
        .form(&params)
        .timeout(Duration::from_secs(10))
        .send()
        .await
        .map_err(|e| {
            if e.is_timeout() {
                UspsError::Timeout
            } else {
                UspsError::Other(e.to_string())
            }
        })?;

    match resp.status().as_u16() {
        200 => {
            let body: TokenResponse = resp.json().await.map_err(|e| UspsError::Other(e.to_string()))?;
            let expires_at = Instant::now() + Duration::from_secs(body.expires_in.saturating_sub(30));
            Ok(CachedToken { token: body.access_token, expires_at })
        }
        401 => Err(UspsError::Unauthorized),
        429 => Err(UspsError::RateLimit),
        503 => Err(UspsError::ServiceUnavailable),
        _ => Err(UspsError::Other(format!("OAuth token request failed: {}", resp.status()))),
    }
}

// ── Token Cache Helper ─────────────────────────────────────────────────────

/// Get or refresh the cached USPS token.
pub async fn get_token(
    cache: &std::sync::Arc<std::sync::Mutex<Option<CachedToken>>>,
    client: &Client,
    client_id: &str,
    client_secret: &str,
) -> Result<String, UspsError> {
    {
        let guard = cache.lock().unwrap();
        if let Some(ref cached) = *guard {
            if Instant::now() < cached.expires_at {
                return Ok(cached.token.clone());
            }
        }
    }
    // Need to fetch or refresh
    let new_token = fetch_token(client, client_id, client_secret).await?;
    let token_str = new_token.token.clone();
    let mut guard = cache.lock().unwrap();
    *guard = Some(new_token);
    Ok(token_str)
}

// ── Tracking Request ───────────────────────────────────────────────────────

pub async fn fetch_tracking(
    client: &Client,
    token: &str,
    tracking_number: &str,
) -> Result<TrackingDetail, UspsError> {
    let body = vec![TrackingRequestBody {
        tracking_number: tracking_number.to_string(),
    }];

    let resp = client
        .post("https://apis.usps.com/tracking/v3r2/tracking")
        .bearer_auth(token)
        .json(&body)
        .timeout(Duration::from_secs(10))
        .send()
        .await
        .map_err(|e| {
            if e.is_timeout() {
                UspsError::Timeout
            } else {
                UspsError::Other(e.to_string())
            }
        })?;

    match resp.status().as_u16() {
        200 => {
            let mut details: Vec<TrackingDetail> = resp
                .json()
                .await
                .map_err(|e| UspsError::Other(e.to_string()))?;
            details
                .pop()
                .ok_or_else(|| UspsError::Other("Empty tracking response".into()))
        }
        207 => {
            // Try to extract error message from FailureResponse
            let failures: Vec<FailureResponse> = resp
                .json()
                .await
                .unwrap_or_default();
            let msg = failures
                .into_iter()
                .next()
                .and_then(|f| f.error)
                .and_then(|e| e.message)
                .unwrap_or_else(|| "Tracking number not found or invalid.".to_string());
            Err(UspsError::NotFoundOrInvalid(msg))
        }
        400 => Err(UspsError::BadRequest),
        401 => Err(UspsError::Unauthorized),
        429 => Err(UspsError::RateLimit),
        503 => Err(UspsError::ServiceUnavailable),
        _ => Err(UspsError::Other(format!("USPS tracking returned: {}", resp.status()))),
    }
}
