use crate::models::{FeedItem, FreshRSSResponse, SubscriptionList};
use std::time::Duration;

#[derive(Clone)]
pub struct FreshRSSConfig {
    pub url: String,
    pub username: String,
    pub password: String,
}

impl FreshRSSConfig {
    pub fn from_env() -> Option<Self> {
        Some(Self {
            url: std::env::var("FRESHRSS_URL").ok()?,
            username: std::env::var("FRESHRSS_USERNAME").ok()?,
            password: std::env::var("FRESHRSS_PASSWORD").ok()?,
        })
    }
}

struct FreshRSSClient {
    client: reqwest::Client,
    base_url: String,
    token: String,
}

impl FreshRSSClient {
    async fn new(config: &FreshRSSConfig) -> Result<Self, String> {
        let client = build_client();
        let auth_url = format!(
            "{}/api/greader.php/accounts/ClientLogin?Email={}&Passwd={}",
            config.url, config.username, config.password
        );

        let text = client
            .get(&auth_url)
            .send()
            .await
            .map_err(|e| format!("Auth request failed: {e}"))?
            .text()
            .await
            .map_err(|e| format!("Failed to read auth response: {e}"))?;

        let token = text
            .lines()
            .find_map(|line| line.strip_prefix("Auth="))
            .map(|t| t.trim().to_string())
            .ok_or_else(|| "Authentication failed: no Auth token found".to_string())?;

        Ok(Self {
            client,
            base_url: config.url.clone(),
            token,
        })
    }

    fn api_url(&self, path: &str) -> String {
        format!("{}/api/greader.php/{}", self.base_url, path)
    }

    fn auth_get(&self, path: &str) -> reqwest::RequestBuilder {
        self.client
            .get(self.api_url(path))
            .header("Authorization", format!("GoogleLogin auth={}", self.token))
    }

    fn auth_post(&self, path: &str) -> reqwest::RequestBuilder {
        self.client
            .post(self.api_url(path))
            .header("Authorization", format!("GoogleLogin auth={}", self.token))
    }

    async fn fetch_items(&self) -> Result<Vec<FeedItem>, String> {
        let data: FreshRSSResponse = self
            .auth_get("reader/api/0/stream/contents/reading-list?n=60&r=d")
            .send()
            .await
            .map_err(|e| format!("Failed to fetch reading list: {e}"))?
            .json()
            .await
            .map_err(|e| format!("Failed to parse FreshRSS response: {e}"))?;

        let mut items: Vec<FeedItem> = data
            .items
            .iter()
            .map(|item| {
                let link = item
                    .canonical
                    .as_ref()
                    .and_then(|c| c.first())
                    .map(|l| l.href.clone())
                    .unwrap_or_default();

                FeedItem {
                    id: item.id.clone(),
                    title: item.title.clone(),
                    published: item.published,
                    author: item.origin.title.clone(),
                    link,
                    origin: item.origin.title.clone(),
                }
            })
            .collect();

        items.sort_by(|a, b| b.published.cmp(&a.published));
        Ok(items)
    }

    async fn fetch_subscriptions(&self) -> Result<SubscriptionList, String> {
        let response = self
            .auth_get("reader/api/0/subscription/list?output=json")
            .send()
            .await
            .map_err(|e| format!("Failed to fetch subscriptions: {e}"))?;

        if !response.status().is_success() {
            return Err(format!("FreshRSS API error: {}", response.status()));
        }

        response
            .json()
            .await
            .map_err(|e| format!("Failed to parse subscription list: {e}"))
    }

    async fn add_subscription(&self, feed_url: &str) -> Result<String, String> {
        let response = self
            .auth_post("reader/api/0/subscription/quickadd")
            .form(&[("quickadd", feed_url)])
            .send()
            .await
            .map_err(|e| format!("Failed to add subscription: {e}"))?;

        if !response.status().is_success() {
            let status = response.status();
            let body = response.text().await.unwrap_or_default();
            return Err(format!("FreshRSS API error ({}): {}", status, body));
        }

        let stream_id = format!("feed/{feed_url}");
        let response = self
            .auth_post("reader/api/0/subscription/edit")
            .form(&[
                ("ac", "edit"),
                ("s", &stream_id),
                ("a", "user/-/label/Feeds"),
            ])
            .send()
            .await
            .map_err(|e| format!("Feed added but failed to set category: {e}"))?;

        if !response.status().is_success() {
            let status = response.status();
            let body = response.text().await.unwrap_or_default();
            return Err(format!(
                "Feed added but failed to set category ({}): {}",
                status, body
            ));
        }

        Ok(format!("Successfully added feed: {feed_url}"))
    }
}

fn build_client() -> reqwest::Client {
    reqwest::Client::builder()
        .timeout(Duration::from_secs(5))
        .build()
        .expect("Failed to build HTTP client")
}

async fn fetch_feed_from_url(client: &reqwest::Client, url: &str) -> Vec<FeedItem> {
    let response = match client.get(url).send().await {
        Ok(r) => r,
        Err(e) => {
            eprintln!("Failed to fetch feed {url}: {e}");
            return Vec::new();
        }
    };

    let body = match response.bytes().await {
        Ok(b) => b,
        Err(e) => {
            eprintln!("Failed to read feed body {url}: {e}");
            return Vec::new();
        }
    };

    let feed = match feed_rs::parser::parse(&body[..]) {
        Ok(f) => f,
        Err(e) => {
            eprintln!("Failed to parse feed {url}: {e}");
            return Vec::new();
        }
    };

    let feed_title = feed
        .title
        .as_ref()
        .map(|t| t.content.clone())
        .unwrap_or_default();

    feed.entries
        .iter()
        .map(|entry| {
            let published = entry
                .published
                .or(entry.updated)
                .map(|dt| dt.timestamp())
                .unwrap_or(0);

            let link = entry
                .links
                .first()
                .map(|l| l.href.clone())
                .unwrap_or_default();

            let title = entry
                .title
                .as_ref()
                .map(|t| t.content.clone())
                .unwrap_or_default();

            let id = entry.id.clone();

            let entry_author = entry
                .authors
                .first()
                .map(|a| a.name.clone())
                .unwrap_or_default();

            let author = if entry_author.is_empty() {
                feed_title.clone()
            } else {
                format!("{} - {}", feed_title, entry_author)
            };

            FeedItem {
                id,
                title,
                published,
                author,
                link,
                origin: feed_title.clone(),
            }
        })
        .collect()
}

pub async fn parse_urls(urls: &[String]) -> Vec<FeedItem> {
    let client = build_client();
    let mut handles = Vec::new();

    for url in urls {
        let client = client.clone();
        let url = url.clone();
        handles.push(tokio::spawn(async move {
            fetch_feed_from_url(&client, &url).await
        }));
    }

    let mut all_items = Vec::new();
    for handle in handles {
        if let Ok(items) = handle.await {
            all_items.extend(items);
        }
    }

    all_items.sort_by(|a, b| b.published.cmp(&a.published));
    all_items
}

pub fn parse_opml(content: &str) -> Vec<String> {
    let mut urls = Vec::new();
    let mut reader = quick_xml::Reader::from_str(content);

    loop {
        match reader.read_event() {
            Ok(quick_xml::events::Event::Empty(ref e))
            | Ok(quick_xml::events::Event::Start(ref e)) => {
                if e.name().as_ref() == b"outline" {
                    for attr in e.attributes().flatten() {
                        if attr.key.as_ref() == b"xmlUrl" {
                            if let Ok(val) = attr.decode_and_unescape_value(reader.decoder()) {
                                let url = val.to_string();
                                if !url.is_empty() {
                                    urls.push(url);
                                }
                            }
                        }
                    }
                }
            }
            Ok(quick_xml::events::Event::Eof) => break,
            Err(e) => {
                eprintln!("Error parsing OPML: {e}");
                break;
            }
            _ => {}
        }
    }

    urls
}

pub async fn fetch_freshrss_items(config: &FreshRSSConfig) -> Result<Vec<FeedItem>, String> {
    FreshRSSClient::new(config).await?.fetch_items().await
}

pub async fn fetch_freshrss_subscriptions(
    config: &FreshRSSConfig,
) -> Result<SubscriptionList, String> {
    FreshRSSClient::new(config).await?.fetch_subscriptions().await
}

pub async fn add_freshrss_subscription(
    config: &FreshRSSConfig,
    feed_url: &str,
) -> Result<String, String> {
    FreshRSSClient::new(config).await?.add_subscription(feed_url).await
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn parse_opml_extracts_xml_urls() {
        let opml = r#"<?xml version="1.0" encoding="UTF-8"?>
<opml version="2.0">
  <body>
    <outline type="rss" text="Blog A" xmlUrl="https://a.com/feed" />
    <outline type="rss" text="Blog B" xmlUrl="https://b.com/rss" />
  </body>
</opml>"#;
        let urls = parse_opml(opml);
        assert_eq!(urls, vec!["https://a.com/feed", "https://b.com/rss"]);
    }

    #[test]
    fn parse_opml_empty_document() {
        let opml = r#"<?xml version="1.0"?><opml><body></body></opml>"#;
        assert!(parse_opml(opml).is_empty());
    }

    #[test]
    fn parse_opml_no_xml_url_attribute() {
        let opml = r#"<?xml version="1.0"?>
<opml><body>
  <outline type="rss" text="No URL" htmlUrl="https://example.com" />
</body></opml>"#;
        assert!(parse_opml(opml).is_empty());
    }

    #[test]
    fn parse_opml_nested_outlines() {
        let opml = r#"<?xml version="1.0"?>
<opml><body>
  <outline text="Category">
    <outline type="rss" text="Nested" xmlUrl="https://nested.com/feed" />
  </outline>
</body></opml>"#;
        let urls = parse_opml(opml);
        assert_eq!(urls, vec!["https://nested.com/feed"]);
    }

    #[test]
    fn parse_opml_skips_empty_url() {
        let opml = r#"<?xml version="1.0"?>
<opml><body>
  <outline type="rss" text="Empty" xmlUrl="" />
  <outline type="rss" text="Valid" xmlUrl="https://valid.com/feed" />
</body></opml>"#;
        let urls = parse_opml(opml);
        assert_eq!(urls, vec!["https://valid.com/feed"]);
    }
}

pub async fn get_feed_items(
    url_query: Option<&str>,
    freshrss_config: Option<&FreshRSSConfig>,
) -> Result<(Vec<FeedItem>, Option<Vec<String>>), String> {
    if let Some(query) = url_query {
        let urls: Vec<String> = query
            .split(',')
            .map(|u| u.trim().to_string())
            .filter(|u| !u.is_empty())
            .collect();

        if !urls.is_empty() {
            let items = parse_urls(&urls).await;
            return Ok((items, Some(urls)));
        }
    }

    if let Ok(content) = tokio::fs::read_to_string("feeds.opml").await {
        let urls = parse_opml(&content);
        if !urls.is_empty() {
            let items = parse_urls(&urls).await;
            return Ok((items, None));
        }
    }

    if let Some(config) = freshrss_config {
        let items = fetch_freshrss_items(config).await?;
        return Ok((items, None));
    }

    if let Ok(default_feed) = std::env::var("DEFAULT_FEED") {
        let urls: Vec<String> = default_feed
            .split(',')
            .map(|u| u.trim().to_string())
            .filter(|u| !u.is_empty())
            .collect();

        if !urls.is_empty() {
            let items = parse_urls(&urls).await;
            return Ok((items, Some(urls)));
        }
    }

    Err("No feed source configured. Set FRESHRSS_URL/FRESHRSS_USERNAME/FRESHRSS_PASSWORD or DEFAULT_FEED.".to_string())
}
