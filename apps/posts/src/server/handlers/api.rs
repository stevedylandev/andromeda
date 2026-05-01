use axum::{
    Json,
    extract::{Path, Query, State},
    http::StatusCode,
    response::{IntoResponse, Response},
};
use serde::{Deserialize, Serialize};
use serde_json::json;
use std::sync::Arc;

const DEFAULT_LIST_LIMIT: i64 = 30;

#[derive(Deserialize)]
pub struct ListPostsQuery {
    limit: Option<i64>,
}

use super::super::*;
use crate::db;

#[derive(Serialize)]
struct ApiPostSummary {
    short_id: String,
    title: Option<String>,
    slug: String,
    published_date: Option<String>,
    meta_description: Option<String>,
    meta_image: Option<String>,
    canonical_url: Option<String>,
    lang: String,
    tags: Option<String>,
    content: String,
    created_at: String,
    updated_at: String,
}

#[derive(Serialize)]
struct ApiPostDetail {
    short_id: String,
    title: Option<String>,
    slug: String,
    alias: Option<String>,
    canonical_url: Option<String>,
    published_date: Option<String>,
    meta_description: Option<String>,
    meta_image: Option<String>,
    lang: String,
    tags: Option<String>,
    content: String,
    created_at: String,
    updated_at: String,
}

#[derive(Serialize)]
struct ApiPostsList {
    posts: Vec<ApiPostSummary>,
}

impl From<Post> for ApiPostSummary {
    fn from(p: Post) -> Self {
        Self {
            short_id: p.short_id,
            title: p.title,
            slug: p.slug,
            published_date: p.published_date,
            meta_description: p.meta_description,
            meta_image: p.meta_image,
            canonical_url: p.canonical_url,
            lang: p.lang,
            tags: p.tags,
            content: p.content,
            created_at: p.created_at,
            updated_at: p.updated_at,
        }
    }
}

impl From<Post> for ApiPostDetail {
    fn from(p: Post) -> Self {
        Self {
            short_id: p.short_id,
            title: p.title,
            slug: p.slug,
            alias: p.alias,
            canonical_url: p.canonical_url,
            published_date: p.published_date,
            meta_description: p.meta_description,
            meta_image: p.meta_image,
            lang: p.lang,
            tags: p.tags,
            content: p.content,
            created_at: p.created_at,
            updated_at: p.updated_at,
        }
    }
}

pub async fn list_posts(
    State(state): State<Arc<AppState>>,
    Query(params): Query<ListPostsQuery>,
) -> Response {
    let limit = params.limit.unwrap_or(DEFAULT_LIST_LIMIT).max(0);
    match db::get_published_posts(&state.db, Some(limit)) {
        Ok(posts) => {
            let posts = posts.into_iter().map(ApiPostSummary::from).collect();
            Json(ApiPostsList { posts }).into_response()
        }
        Err(e) => {
            tracing::error!("Failed to list posts for API: {}", e);
            (
                StatusCode::INTERNAL_SERVER_ERROR,
                Json(json!({ "error": "internal server error" })),
            )
                .into_response()
        }
    }
}

pub async fn get_post(
    State(state): State<Arc<AppState>>,
    Path(slug): Path<String>,
) -> Response {
    match db::get_post_by_slug(&state.db, &slug) {
        Ok(Some(post)) if post.status == "published" => {
            Json(ApiPostDetail::from(post)).into_response()
        }
        Ok(_) => (
            StatusCode::NOT_FOUND,
            Json(json!({ "error": "not found" })),
        )
            .into_response(),
        Err(e) => {
            tracing::error!("Failed to get post for API: {}", e);
            (
                StatusCode::INTERNAL_SERVER_ERROR,
                Json(json!({ "error": "internal server error" })),
            )
                .into_response()
        }
    }
}
