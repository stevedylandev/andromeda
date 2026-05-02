use scraper::{Html, Selector};
use std::time::Duration;
use url::Url;

fn build_client() -> reqwest::Client {
    reqwest::Client::builder()
        .timeout(Duration::from_secs(15))
        .user_agent("andromeda-bookmarks/0.1 (+https://github.com/stevedylandev/andromeda)")
        .build()
        .expect("Failed to build HTTP client")
}

/// Best-effort favicon URL for a page. Parses `<link rel="icon">` from the
/// HTML, falls back to `/favicon.ico` at the site root. Returns None if
/// the URL is invalid.
pub async fn discover_favicon(page_url: &str) -> Option<String> {
    let parsed = Url::parse(page_url).ok()?;
    let client = build_client();

    if let Ok(resp) = client.get(page_url).send().await {
        if let Ok(body) = resp.text().await {
            let document = Html::parse_document(&body);
            let selector = Selector::parse(
                r#"link[rel="icon"], link[rel="shortcut icon"], link[rel="apple-touch-icon"]"#,
            )
            .ok()?;
            if let Some(href) = document
                .select(&selector)
                .find_map(|el| el.attr("href"))
            {
                if let Ok(resolved) = parsed.join(href) {
                    return Some(resolved.to_string());
                }
            }
        }
    }

    parsed.join("/favicon.ico").ok().map(|u| u.to_string())
}
