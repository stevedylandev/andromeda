use crate::models::FeedItem;
use quick_xml::events::Event;
use scraper::{Html, Selector};
use std::time::Duration;
use url::Url;

/// One outline entry from an OPML document (subscription plus optional category name).
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct OpmlEntry {
    pub xml_url: String,
    pub title: Option<String>,
    pub html_url: Option<String>,
    pub category: Option<String>,
}

/// Result of a conditional fetch against an RSS/Atom feed.
#[derive(Debug)]
pub struct FetchResult {
    /// HTTP status code. 304 means nothing changed; items will be empty.
    pub status: u16,
    pub etag: Option<String>,
    pub last_modified: Option<String>,
    pub title: Option<String>,
    pub site_url: Option<String>,
    pub entries: Vec<ParsedEntry>,
}

#[derive(Debug, Clone)]
pub struct ParsedEntry {
    pub guid: String,
    pub title: String,
    pub link: String,
    pub author: Option<String>,
    pub published_at: i64,
}

const DERIVED_TITLE_MAX_CHARS: usize = 80;

/// Build a synthetic title from an entry's HTML description when the feed
/// publishes empty `<title>` tags (common for Micro.blog-style microposts).
/// Strips tags, collapses whitespace, and truncates to a readable preview.
fn derive_title_from_html(html: &str) -> String {
    let fragment = Html::parse_fragment(html);
    let text: String = fragment.root_element().text().collect();
    let collapsed = text.split_whitespace().collect::<Vec<_>>().join(" ");
    let mut chars = collapsed.chars();
    let truncated: String = chars.by_ref().take(DERIVED_TITLE_MAX_CHARS).collect();
    if chars.next().is_some() {
        format!("{}…", truncated.trim_end())
    } else {
        truncated
    }
}

fn build_client() -> reqwest::Client {
    reqwest::Client::builder()
        .timeout(Duration::from_secs(15))
        .user_agent("andromeda-feeds/0.1 (+https://github.com/stevedylandev/andromeda)")
        .build()
        .expect("Failed to build HTTP client")
}

pub async fn fetch_feed(
    url: &str,
    etag: Option<&str>,
    last_modified: Option<&str>,
) -> Result<FetchResult, String> {
    let client = build_client();
    let mut req = client.get(url);
    if let Some(tag) = etag {
        req = req.header("If-None-Match", tag);
    }
    if let Some(lm) = last_modified {
        req = req.header("If-Modified-Since", lm);
    }

    let resp = req.send().await.map_err(|e| format!("fetch failed: {e}"))?;
    let status = resp.status().as_u16();
    let headers = resp.headers().clone();
    let new_etag = headers
        .get(reqwest::header::ETAG)
        .and_then(|v| v.to_str().ok())
        .map(|s| s.to_string());
    let new_last_modified = headers
        .get(reqwest::header::LAST_MODIFIED)
        .and_then(|v| v.to_str().ok())
        .map(|s| s.to_string());

    if status == 304 {
        return Ok(FetchResult {
            status,
            etag: new_etag.or_else(|| etag.map(|s| s.to_string())),
            last_modified: new_last_modified.or_else(|| last_modified.map(|s| s.to_string())),
            title: None,
            site_url: None,
            entries: Vec::new(),
        });
    }

    if !resp.status().is_success() {
        return Err(format!("upstream returned {status}"));
    }

    let body = resp
        .bytes()
        .await
        .map_err(|e| format!("read body failed: {e}"))?;
    let feed =
        feed_rs::parser::parse(&body[..]).map_err(|e| format!("feed parse failed: {e}"))?;

    let title = feed.title.as_ref().map(|t| t.content.clone());
    let site_url = feed
        .links
        .iter()
        .find(|l| l.rel.as_deref() != Some("self"))
        .map(|l| l.href.clone())
        .or_else(|| feed.links.first().map(|l| l.href.clone()));

    let entries = feed
        .entries
        .into_iter()
        .map(|entry| {
            let published_at = entry
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
                .filter(|t| !t.trim().is_empty())
                .or_else(|| {
                    let html = entry
                        .summary
                        .as_ref()
                        .map(|s| s.content.as_str())
                        .or_else(|| entry.content.as_ref().and_then(|c| c.body.as_deref()))?;
                    let derived = derive_title_from_html(html);
                    if derived.is_empty() {
                        None
                    } else {
                        Some(derived)
                    }
                })
                .unwrap_or_default();
            let author = entry.authors.first().map(|a| a.name.clone());
            let guid = if !entry.id.is_empty() {
                entry.id
            } else {
                link.clone()
            };
            ParsedEntry {
                guid,
                title,
                link,
                author,
                published_at,
            }
        })
        .collect();

    Ok(FetchResult {
        status,
        etag: new_etag,
        last_modified: new_last_modified,
        title,
        site_url,
        entries,
    })
}

/// Ad-hoc preview: parse one or more feed URLs and return flattened items.
/// Kept for the `?url=` bypass mode on the index page.
pub async fn preview_urls(urls: &[String]) -> Vec<FeedItem> {
    let mut handles = Vec::new();
    for url in urls {
        let url = url.clone();
        handles.push(tokio::spawn(async move {
            let result = fetch_feed(&url, None, None).await;
            match result {
                Ok(r) => {
                    let feed_title = r.title.clone().unwrap_or_default();
                    r.entries
                        .into_iter()
                        .map(|e| FeedItem {
                            title: e.title,
                            link: e.link,
                            published: e.published_at,
                            author: match e.author {
                                Some(a) if !a.is_empty() && !feed_title.is_empty() => {
                                    format!("{feed_title} - {a}")
                                }
                                Some(a) if !a.is_empty() => a,
                                _ => feed_title.clone(),
                            },
                        })
                        .collect::<Vec<_>>()
                }
                Err(e) => {
                    tracing::warn!("preview fetch failed for {url}: {e}");
                    Vec::new()
                }
            }
        }));
    }

    let mut all = Vec::new();
    for h in handles {
        if let Ok(items) = h.await {
            all.extend(items);
        }
    }
    all.sort_by(|a, b| b.published.cmp(&a.published));
    all
}

/// Parse an OPML document into outline entries, carrying the parent `<outline>` title
/// as a category when the parent has no `xmlUrl`.
pub fn parse_opml(content: &str) -> Vec<OpmlEntry> {
    let mut entries = Vec::new();
    let mut reader = quick_xml::Reader::from_str(content);
    let mut category_stack: Vec<String> = Vec::new();

    loop {
        match reader.read_event() {
            Ok(Event::Start(ref e)) if e.name().as_ref() == b"outline" => {
                let mut xml_url: Option<String> = None;
                let mut title: Option<String> = None;
                let mut html_url: Option<String> = None;
                for attr in e.attributes().flatten() {
                    let val = attr
                        .decode_and_unescape_value(reader.decoder())
                        .ok()
                        .map(|v| v.to_string());
                    match attr.key.as_ref() {
                        b"xmlUrl" => xml_url = val.filter(|v| !v.is_empty()),
                        b"title" => title = val,
                        b"text" if title.is_none() => title = val,
                        b"htmlUrl" => html_url = val,
                        _ => {}
                    }
                }

                if let Some(xml) = xml_url {
                    entries.push(OpmlEntry {
                        xml_url: xml,
                        title,
                        html_url,
                        category: category_stack.last().cloned(),
                    });
                    category_stack.push(String::new()); // balance Close event
                } else {
                    category_stack.push(title.unwrap_or_default());
                }
            }
            Ok(Event::Empty(ref e)) if e.name().as_ref() == b"outline" => {
                let mut xml_url: Option<String> = None;
                let mut title: Option<String> = None;
                let mut html_url: Option<String> = None;
                for attr in e.attributes().flatten() {
                    let val = attr
                        .decode_and_unescape_value(reader.decoder())
                        .ok()
                        .map(|v| v.to_string());
                    match attr.key.as_ref() {
                        b"xmlUrl" => xml_url = val.filter(|v| !v.is_empty()),
                        b"title" => title = val,
                        b"text" if title.is_none() => title = val,
                        b"htmlUrl" => html_url = val,
                        _ => {}
                    }
                }
                if let Some(xml) = xml_url {
                    entries.push(OpmlEntry {
                        xml_url: xml,
                        title,
                        html_url,
                        category: category_stack.last().cloned().filter(|c| !c.is_empty()),
                    });
                }
            }
            Ok(Event::End(ref e)) if e.name().as_ref() == b"outline" => {
                category_stack.pop();
            }
            Ok(Event::Eof) => break,
            Err(e) => {
                tracing::warn!("OPML parse error: {e}");
                break;
            }
            _ => {}
        }
    }

    entries
}

pub async fn discover_feeds(base_url: &str) -> Result<Vec<String>, String> {
    let parsed = Url::parse(base_url).map_err(|e| format!("Invalid URL: {e}"))?;
    let client = build_client();
    let mut feeds = Vec::new();

    if let Ok(response) = client.get(base_url).send().await {
        if let Ok(body) = response.text().await {
            let document = Html::parse_document(&body);
            let selector = Selector::parse(r#"link[rel="alternate"]"#).unwrap();
            for element in document.select(&selector) {
                let type_attr = element.attr("type").unwrap_or_default();
                if type_attr.contains("rss")
                    || type_attr.contains("atom")
                    || type_attr.contains("xml")
                {
                    if let Some(href) = element.attr("href") {
                        let resolved = parsed
                            .join(href)
                            .map(|u| u.to_string())
                            .unwrap_or_else(|_| href.to_string());
                        if !feeds.contains(&resolved) {
                            feeds.push(resolved);
                        }
                    }
                }
            }
        }
    }

    if feeds.is_empty() {
        let common_paths = [
            "/feed",
            "/feed.xml",
            "/rss",
            "/rss.xml",
            "/atom.xml",
            "/index.xml",
            "/feed/rss",
            "/blog/feed",
            "/blog/rss",
        ];
        let mut handles = Vec::new();
        for path in common_paths {
            let probe_url = match parsed.join(path) {
                Ok(u) => u.to_string(),
                Err(_) => continue,
            };
            let client = client.clone();
            handles.push(tokio::spawn(async move {
                if let Ok(resp) = client.head(&probe_url).send().await {
                    if resp.status().is_success() {
                        if let Some(ct) = resp.headers().get("content-type") {
                            let ct = ct.to_str().unwrap_or_default();
                            if ct.contains("xml") || ct.contains("rss") || ct.contains("atom") {
                                return Some(probe_url);
                            }
                        }
                    }
                }
                None
            }));
        }
        for h in handles {
            if let Ok(Some(url)) = h.await {
                if !feeds.contains(&url) {
                    feeds.push(url);
                }
            }
        }
    }

    if feeds.is_empty() {
        Err("No feeds found at this URL".to_string())
    } else {
        Ok(feeds)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn derive_title_strips_html_and_collapses_whitespace() {
        let html = "<p>If they launched   full-time\n\ngoblin mode, I&rsquo;d use it</p>";
        assert_eq!(
            derive_title_from_html(html),
            "If they launched full-time goblin mode, I\u{2019}d use it"
        );
    }

    #[test]
    fn derive_title_truncates_long_text() {
        let html = format!("<p>{}</p>", "a ".repeat(100));
        let out = derive_title_from_html(&html);
        assert!(out.ends_with('…'));
        assert!(out.chars().count() <= DERIVED_TITLE_MAX_CHARS + 1);
    }

    #[test]
    fn derive_title_empty_html_yields_empty() {
        assert_eq!(derive_title_from_html(""), "");
        assert_eq!(derive_title_from_html("<p>   </p>"), "");
    }

    #[test]
    fn parse_opml_flat_outlines() {
        let opml = r#"<?xml version="1.0" encoding="UTF-8"?>
<opml version="2.0"><body>
    <outline type="rss" text="Blog A" xmlUrl="https://a.com/feed" />
    <outline type="rss" text="Blog B" xmlUrl="https://b.com/rss" />
</body></opml>"#;
        let entries = parse_opml(opml);
        assert_eq!(entries.len(), 2);
        assert_eq!(entries[0].xml_url, "https://a.com/feed");
        assert_eq!(entries[0].title.as_deref(), Some("Blog A"));
        assert!(entries[0].category.is_none());
    }

    #[test]
    fn parse_opml_empty() {
        let opml = r#"<?xml version="1.0"?><opml><body></body></opml>"#;
        assert!(parse_opml(opml).is_empty());
    }

    #[test]
    fn parse_opml_no_xml_url_skipped() {
        let opml = r#"<?xml version="1.0"?>
<opml><body><outline type="rss" text="No URL" htmlUrl="https://example.com" /></body></opml>"#;
        assert!(parse_opml(opml).is_empty());
    }

    #[test]
    fn parse_opml_nested_carries_category() {
        let opml = r#"<?xml version="1.0"?>
<opml><body>
  <outline text="Tech">
    <outline type="rss" text="Inner" xmlUrl="https://inner.com/feed" />
  </outline>
</body></opml>"#;
        let entries = parse_opml(opml);
        assert_eq!(entries.len(), 1);
        assert_eq!(entries[0].category.as_deref(), Some("Tech"));
    }

    #[test]
    fn parse_opml_deeply_nested() {
        let opml = r#"<?xml version="1.0"?>
<opml><body>
  <outline text="Root">
    <outline text="Tech">
      <outline type="rss" text="A" xmlUrl="https://a.com/feed" />
    </outline>
    <outline type="rss" text="B" xmlUrl="https://b.com/feed" />
  </outline>
</body></opml>"#;
        let entries = parse_opml(opml);
        assert_eq!(entries.len(), 2);
        assert_eq!(entries[0].xml_url, "https://a.com/feed");
        assert_eq!(entries[0].category.as_deref(), Some("Tech"));
        assert_eq!(entries[1].xml_url, "https://b.com/feed");
        assert_eq!(entries[1].category.as_deref(), Some("Root"));
    }

    #[test]
    fn parse_opml_skips_empty_url() {
        let opml = r#"<?xml version="1.0"?>
<opml><body>
  <outline type="rss" text="Empty" xmlUrl="" />
  <outline type="rss" text="Valid" xmlUrl="https://valid.com/feed" />
</body></opml>"#;
        let entries = parse_opml(opml);
        assert_eq!(entries.len(), 1);
        assert_eq!(entries[0].xml_url, "https://valid.com/feed");
    }
}
