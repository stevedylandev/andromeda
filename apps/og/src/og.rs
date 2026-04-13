use scraper::{Html, Selector};
use std::collections::HashMap;
use url::Url;

pub struct LinkTag {
    pub rel: String,
    pub href: String,
    pub extra: String,
}

pub struct OgResult {
    pub og_tags: HashMap<String, String>,
    pub favicon: Option<String>,
    pub link_tags: Vec<LinkTag>,
}

pub async fn fetch_og_data(target_url: &str) -> Result<OgResult, String> {
    let parsed_url = Url::parse(target_url).map_err(|e| format!("Invalid URL: {e}"))?;

    let client = reqwest::Client::builder()
        .timeout(std::time::Duration::from_secs(10))
        .user_agent("Mozilla/5.0 (compatible; OGPreview/1.0)")
        .build()
        .map_err(|e| format!("Failed to create HTTP client: {e}"))?;

    let resp = client
        .get(target_url)
        .send()
        .await
        .map_err(|e| format!("Failed to fetch URL: {e}"))?;

    if !resp.status().is_success() {
        return Err(format!("HTTP error: {}", resp.status()));
    }

    let content_type = resp
        .headers()
        .get("content-type")
        .and_then(|v| v.to_str().ok())
        .unwrap_or("");
    if !content_type.contains("text/html") && !content_type.contains("application/xhtml") {
        return Err(format!("Not an HTML page (Content-Type: {content_type})"));
    }

    let body = resp
        .text()
        .await
        .map_err(|e| format!("Failed to read response body: {e}"))?;

    let document = Html::parse_document(&body);

    // Extract OG tags
    let mut og_tags = HashMap::new();
    let og_property = Selector::parse(r#"meta[property^="og:"]"#).unwrap();
    let og_name = Selector::parse(r#"meta[name^="og:"]"#).unwrap();

    for el in document.select(&og_property).chain(document.select(&og_name)) {
        let key = el
            .value()
            .attr("property")
            .or_else(|| el.value().attr("name"));
        let value = el.value().attr("content");
        if let (Some(k), Some(v)) = (key, value) {
            og_tags.entry(k.to_string()).or_insert_with(|| v.to_string());
        }
    }

    // Resolve relative og:image URL
    if let Some(image) = og_tags.get("og:image") {
        if let Ok(resolved) = parsed_url.join(image) {
            og_tags.insert("og:image".to_string(), resolved.to_string());
        }
    }

    // Extract favicon
    let favicon = extract_favicon(&document, &parsed_url);

    // Extract head <link> tags
    let link_tags = extract_link_tags(&document, &parsed_url);

    Ok(OgResult { og_tags, favicon, link_tags })
}

fn extract_favicon(document: &Html, base_url: &Url) -> Option<String> {
    let selectors = [
        r#"link[rel="icon"]"#,
        r#"link[rel="shortcut icon"]"#,
        r#"link[rel="apple-touch-icon"]"#,
    ];
    for sel_str in &selectors {
        if let Ok(sel) = Selector::parse(sel_str) {
            if let Some(el) = document.select(&sel).next() {
                if let Some(href) = el.value().attr("href") {
                    if let Ok(resolved) = base_url.join(href) {
                        return Some(resolved.to_string());
                    }
                }
            }
        }
    }
    // Fallback to /favicon.ico
    base_url.join("/favicon.ico").ok().map(|u| u.to_string())
}

fn extract_link_tags(document: &Html, base_url: &Url) -> Vec<LinkTag> {
    let mut link_tags = Vec::new();
    if let Ok(sel) = Selector::parse("head link") {
        for el in document.select(&sel) {
            let rel = el.value().attr("rel").unwrap_or("").to_string();
            let href = el
                .value()
                .attr("href")
                .map(|h| base_url.join(h).map(|u| u.to_string()).unwrap_or_else(|_| h.to_string()))
                .unwrap_or_default();
            let mut extras = Vec::new();
            for (name, val) in el.value().attrs() {
                if name != "rel" && name != "href" {
                    extras.push(format!("{name}=\"{val}\""));
                }
            }
            link_tags.push(LinkTag {
                rel,
                href,
                extra: extras.join(" "),
            });
        }
    }
    link_tags
}

#[cfg(test)]
mod tests {
    use super::*;

    fn base() -> Url {
        Url::parse("https://example.com/page").unwrap()
    }

    // ── extract_favicon ────────────────────────────────────────────────

    #[test]
    fn favicon_from_rel_icon() {
        let doc = Html::parse_document(
            r#"<html><head><link rel="icon" href="/favicon.png"></head></html>"#,
        );
        let result = extract_favicon(&doc, &base());
        assert_eq!(result, Some("https://example.com/favicon.png".to_string()));
    }

    #[test]
    fn favicon_from_shortcut_icon() {
        let doc = Html::parse_document(
            r#"<html><head><link rel="shortcut icon" href="/icon.ico"></head></html>"#,
        );
        let result = extract_favicon(&doc, &base());
        assert_eq!(result, Some("https://example.com/icon.ico".to_string()));
    }

    #[test]
    fn favicon_from_apple_touch_icon() {
        let doc = Html::parse_document(
            r#"<html><head><link rel="apple-touch-icon" href="/apple.png"></head></html>"#,
        );
        let result = extract_favicon(&doc, &base());
        assert_eq!(result, Some("https://example.com/apple.png".to_string()));
    }

    #[test]
    fn favicon_priority_icon_over_shortcut() {
        let doc = Html::parse_document(
            r#"<html><head>
                <link rel="icon" href="/first.png">
                <link rel="shortcut icon" href="/second.ico">
            </head></html>"#,
        );
        let result = extract_favicon(&doc, &base());
        assert_eq!(result, Some("https://example.com/first.png".to_string()));
    }

    #[test]
    fn favicon_fallback_to_favicon_ico() {
        let doc = Html::parse_document("<html><head></head></html>");
        let result = extract_favicon(&doc, &base());
        assert_eq!(result, Some("https://example.com/favicon.ico".to_string()));
    }

    #[test]
    fn favicon_resolves_relative_url() {
        let doc = Html::parse_document(
            r#"<html><head><link rel="icon" href="assets/icon.png"></head></html>"#,
        );
        let result = extract_favicon(&doc, &base());
        assert_eq!(
            result,
            Some("https://example.com/assets/icon.png".to_string())
        );
    }

    // ── extract_link_tags ──────────────────────────────────────────────

    #[test]
    fn link_tags_extracts_multiple() {
        let doc = Html::parse_document(
            r#"<html><head>
                <link rel="stylesheet" href="/style.css">
                <link rel="canonical" href="https://example.com/">
            </head></html>"#,
        );
        let tags = extract_link_tags(&doc, &base());
        assert_eq!(tags.len(), 2);
        assert_eq!(tags[0].rel, "stylesheet");
        assert_eq!(tags[1].rel, "canonical");
    }

    #[test]
    fn link_tags_resolves_relative_href() {
        let doc = Html::parse_document(
            r#"<html><head><link rel="stylesheet" href="css/main.css"></head></html>"#,
        );
        let tags = extract_link_tags(&doc, &base());
        assert_eq!(tags[0].href, "https://example.com/css/main.css");
    }

    #[test]
    fn link_tags_preserves_extra_attrs() {
        let doc = Html::parse_document(
            r#"<html><head><link rel="stylesheet" href="/s.css" type="text/css"></head></html>"#,
        );
        let tags = extract_link_tags(&doc, &base());
        assert!(tags[0].extra.contains("type=\"text/css\""));
    }

    #[test]
    fn link_tags_empty_head() {
        let doc = Html::parse_document("<html><head></head></html>");
        let tags = extract_link_tags(&doc, &base());
        assert!(tags.is_empty());
    }
}
