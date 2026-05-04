use std::time::Duration;

use rusty_s3::actions::S3Action;
use rusty_s3::{Bucket, Credentials, UrlStyle, actions};

#[derive(Clone)]
pub struct R2Config {
    bucket: Bucket,
    creds: Credentials,
    public_url: String,
    http: reqwest::Client,
}

#[derive(Debug)]
pub enum R2Error {
    Http(reqwest::Error),
    Status(u16, String),
}

impl std::fmt::Display for R2Error {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            R2Error::Http(e) => write!(f, "http error: {}", e),
            R2Error::Status(code, body) => write!(f, "R2 returned {}: {}", code, body),
        }
    }
}

impl std::error::Error for R2Error {}

impl From<reqwest::Error> for R2Error {
    fn from(e: reqwest::Error) -> Self {
        R2Error::Http(e)
    }
}

const SIGN_TTL: Duration = Duration::from_secs(60);

impl R2Config {
    pub fn from_env() -> Option<Self> {
        let account_id = std::env::var("R2_ACCOUNT_ID").ok()?;
        let access_key = std::env::var("R2_ACCESS_KEY_ID").ok()?;
        let secret_key = std::env::var("R2_SECRET_ACCESS_KEY").ok()?;
        let bucket_name = std::env::var("R2_BUCKET").ok()?;
        let public_url = std::env::var("R2_PUBLIC_URL").ok()?;

        let endpoint_str = format!("https://{}.r2.cloudflarestorage.com", account_id);
        let endpoint = match endpoint_str.parse() {
            Ok(u) => u,
            Err(e) => {
                tracing::error!("Invalid R2 endpoint URL: {}", e);
                return None;
            }
        };
        let bucket = match Bucket::new(endpoint, UrlStyle::Path, bucket_name, "auto") {
            Ok(b) => b,
            Err(e) => {
                tracing::error!("Failed to construct R2 bucket: {:?}", e);
                return None;
            }
        };
        let creds = Credentials::new(access_key, secret_key);
        let http = reqwest::Client::builder()
            .timeout(Duration::from_secs(30))
            .build()
            .ok()?;

        Some(Self {
            bucket,
            creds,
            public_url: public_url.trim_end_matches('/').to_string(),
            http,
        })
    }

    pub async fn put_object(
        &self,
        key: &str,
        content_type: &str,
        bytes: Vec<u8>,
    ) -> Result<(), R2Error> {
        let mut action = actions::PutObject::new(&self.bucket, Some(&self.creds), key);
        action.headers_mut().insert("content-type", content_type);
        let url = action.sign(SIGN_TTL);

        let resp = self
            .http
            .put(url)
            .header("content-type", content_type)
            .body(bytes)
            .send()
            .await?;

        if !resp.status().is_success() {
            let code = resp.status().as_u16();
            let body = resp.text().await.unwrap_or_default();
            return Err(R2Error::Status(code, body));
        }
        Ok(())
    }

    pub async fn delete_object(&self, key: &str) -> Result<(), R2Error> {
        let action = actions::DeleteObject::new(&self.bucket, Some(&self.creds), key);
        let url = action.sign(SIGN_TTL);

        let resp = self.http.delete(url).send().await?;
        let status = resp.status();
        if status.is_success() || status.as_u16() == 404 {
            return Ok(());
        }
        let body = resp.text().await.unwrap_or_default();
        Err(R2Error::Status(status.as_u16(), body))
    }

    pub fn public_url_for(&self, key: &str) -> String {
        format!("{}/{}", self.public_url, key)
    }
}
