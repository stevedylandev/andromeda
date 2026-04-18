use serde::{Deserialize, Serialize};

/// Normalized feed entry used by the index template and ad-hoc URL previews.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FeedItem {
    pub title: String,
    pub link: String,
    pub author: String,
    pub published: i64,
}
