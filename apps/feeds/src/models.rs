#![allow(dead_code)]
use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FeedItem {
    pub id: String,
    pub title: String,
    pub published: i64,
    pub author: String,
    pub link: String,
    pub origin: String,
}

#[derive(Debug, Deserialize)]
pub struct FreshRSSResponse {
    pub id: String,
    pub updated: Option<i64>,
    pub items: Vec<FreshRSSItem>,
    pub continuation: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct FreshRSSItem {
    pub id: String,
    pub title: String,
    pub published: i64,
    pub author: Option<String>,
    pub canonical: Option<Vec<FreshRSSLink>>,
    pub origin: FreshRSSOrigin,
}

#[derive(Debug, Deserialize)]
pub struct FreshRSSLink {
    pub href: String,
}

#[derive(Debug, Clone, Deserialize)]
pub struct FreshRSSOrigin {
    #[serde(rename = "streamId")]
    pub stream_id: String,
    #[serde(rename = "htmlUrl")]
    pub html_url: Option<String>,
    pub title: String,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct Subscription {
    pub id: String,
    pub title: String,
    pub url: String,
    #[serde(rename = "htmlUrl", skip_serializing_if = "Option::is_none")]
    pub html_url: Option<String>,
}

#[derive(Debug, Serialize, Deserialize)]
pub struct SubscriptionList {
    pub subscriptions: Option<Vec<Subscription>>,
}
