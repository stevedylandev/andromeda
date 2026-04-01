use crate::models::{FeedItem, FreshRSSResponse, SubscriptionList};
use std::time::Duration;

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

async fn freshrss_auth(
    client: &reqwest::Client,
    freshrss_url: &str,
    username: &str,
    password: &str,
) -> Result<String, String> {
    let auth_url = format!(
        "{}/api/greader.php/accounts/ClientLogin?Email={}&Passwd={}",
        freshrss_url, username, password
    );

    let response = client
        .get(&auth_url)
        .send()
        .await
        .map_err(|e| format!("Auth request failed: {e}"))?;

    let text = response
        .text()
        .await
        .map_err(|e| format!("Failed to read auth response: {e}"))?;

    for line in text.lines() {
        if let Some(token) = line.strip_prefix("Auth=") {
            return Ok(token.trim().to_string());
        }
    }

    Err("Authentication failed: no Auth token found".to_string())
}

pub async fn fetch_freshrss_items(
    freshrss_url: &str,
    username: &str,
    password: &str,
) -> Result<Vec<FeedItem>, String> {
    let client = build_client();
    let token = freshrss_auth(&client, freshrss_url, username, password).await?;

    let url = format!(
        "{}/api/greader.php/reader/api/0/stream/contents/reading-list?n=60&r=d",
        freshrss_url
    );

    let response = client
        .get(&url)
        .header("Authorization", format!("GoogleLogin auth={token}"))
        .send()
        .await
        .map_err(|e| format!("Failed to fetch reading list: {e}"))?;

    let data: FreshRSSResponse = response
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

pub async fn fetch_freshrss_subscriptions(
    freshrss_url: &str,
    username: &str,
    password: &str,
) -> Result<SubscriptionList, String> {
    let client = build_client();
    let token = freshrss_auth(&client, freshrss_url, username, password).await?;

    let url = format!(
        "{}/api/greader.php/reader/api/0/subscription/list?output=json",
        freshrss_url
    );

    let response = client
        .get(&url)
        .header("Authorization", format!("GoogleLogin auth={token}"))
        .send()
        .await
        .map_err(|e| format!("Failed to fetch subscriptions: {e}"))?;

    if !response.status().is_success() {
        return Err(format!("FreshRSS API error: {}", response.status()));
    }

    let data: SubscriptionList = response
        .json()
        .await
        .map_err(|e| format!("Failed to parse subscription list: {e}"))?;

    Ok(data)
}

pub async fn add_freshrss_subscription(
    freshrss_url: &str,
    username: &str,
    password: &str,
    feed_url: &str,
) -> Result<String, String> {
    let client = build_client();
    let token = freshrss_auth(&client, freshrss_url, username, password).await?;

    let url = format!(
        "{}/api/greader.php/reader/api/0/subscription/quickadd",
        freshrss_url
    );

    let response = client
        .post(&url)
        .header("Authorization", format!("GoogleLogin auth={token}"))
        .form(&[("quickadd", feed_url)])
        .send()
        .await
        .map_err(|e| format!("Failed to add subscription: {e}"))?;

    if !response.status().is_success() {
        let status = response.status();
        let body = response.text().await.unwrap_or_default();
        return Err(format!("FreshRSS API error ({}): {}", status, body));
    }

    // Assign the "Feeds" category via subscription/edit
    let edit_url = format!(
        "{}/api/greader.php/reader/api/0/subscription/edit",
        freshrss_url
    );

    let stream_id = format!("feed/{feed_url}");
    let response = client
        .post(&edit_url)
        .header("Authorization", format!("GoogleLogin auth={token}"))
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

pub async fn get_feed_items(
    url_query: Option<&str>,
) -> Result<(Vec<FeedItem>, Option<Vec<String>>), String> {
    // Priority 1: URL query parameter
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

    // Priority 2: Local OPML file
    if let Ok(content) = tokio::fs::read_to_string("feeds.opml").await {
        let urls = parse_opml(&content);
        if !urls.is_empty() {
            let items = parse_urls(&urls).await;
            return Ok((items, None));
        }
    }

    // Priority 3: FreshRSS fallback
    let freshrss_url = std::env::var("FRESHRSS_URL").map_err(|_| "FRESHRSS_URL not set")?;
    let username =
        std::env::var("FRESHRSS_USERNAME").map_err(|_| "FRESHRSS_USERNAME not set")?;
    let password =
        std::env::var("FRESHRSS_PASSWORD").map_err(|_| "FRESHRSS_PASSWORD not set")?;

    let items = fetch_freshrss_items(&freshrss_url, &username, &password).await?;
    Ok((items, None))
}
