use askama_web::WebTemplate;
use axum::{
    Json,
    extract::{Path, State},
    http::{HeaderValue, StatusCode},
    response::{Html, IntoResponse, Response},
};
use std::sync::Arc;

use super::super::*;
use crate::{auth, db};

pub async fn serve_static(Path(path): Path<String>) -> Response {
    match Static::get(&path) {
        Some(file) => {
            let mime = mime_from_path(&path);
            (
                StatusCode::OK,
                [(axum::http::header::CONTENT_TYPE, HeaderValue::from_static(mime))],
                file.data.to_vec(),
            )
                .into_response()
        }
        None => StatusCode::NOT_FOUND.into_response(),
    }
}

pub async fn get_index(State(state): State<Arc<AppState>>) -> Response {
    match db::get_cellar_wines(&state.db) {
        Ok(wines) => {
            let wines: Vec<WineWithSvg> = wines
                .into_iter()
                .map(|wine| {
                    let pentagon_svg = build_pentagon_svg(
                        wine.sweetness,
                        wine.acidity,
                        wine.tannin,
                        wine.alcohol,
                        wine.body,
                        80.0,
                        false,
                    );
                    WineWithSvg { wine, pentagon_svg }
                })
                .collect();
            WebTemplate(IndexTemplate { wines }).into_response()
        }
        Err(e) => {
            tracing::error!("Failed to list wines: {}", e);
            (StatusCode::INTERNAL_SERVER_ERROR, Html("Server error".to_string())).into_response()
        }
    }
}

pub async fn get_wine_detail(
    State(state): State<Arc<AppState>>,
    Path(short_id): Path<String>,
) -> Response {
    match db::get_wine_by_short_id(&state.db, &short_id) {
        Ok(Some(wine)) => {
            let pentagon_svg = build_pentagon_svg(
                wine.sweetness,
                wine.acidity,
                wine.tannin,
                wine.alcohol,
                wine.body,
                250.0,
                true,
            );
            let bars_svg = build_bars_svg(
                wine.clarity,
                wine.color_intensity,
                wine.aroma_intensity,
                wine.nose_complexity,
                250.0,
            );
            WebTemplate(WineDetailTemplate { wine, pentagon_svg, bars_svg }).into_response()
        }
        Ok(None) => (StatusCode::NOT_FOUND, Html("Wine not found".to_string())).into_response(),
        Err(e) => {
            tracing::error!("Failed to get wine: {}", e);
            (StatusCode::INTERNAL_SERVER_ERROR, Html("Server error".to_string())).into_response()
        }
    }
}

pub async fn api_list_wines(State(state): State<Arc<AppState>>) -> Response {
    match db::get_cellar_wines(&state.db) {
        Ok(wines) => Json(wines).into_response(),
        Err(e) => {
            tracing::error!("api_list_wines: {}", e);
            StatusCode::INTERNAL_SERVER_ERROR.into_response()
        }
    }
}

pub async fn api_get_wine(
    State(state): State<Arc<AppState>>,
    Path(short_id): Path<String>,
) -> Response {
    match db::get_wine_by_short_id(&state.db, &short_id) {
        Ok(Some(wine)) => Json(wine).into_response(),
        Ok(None) => StatusCode::NOT_FOUND.into_response(),
        Err(e) => {
            tracing::error!("api_get_wine: {}", e);
            StatusCode::INTERNAL_SERVER_ERROR.into_response()
        }
    }
}

pub async fn get_wine_image(
    State(state): State<Arc<AppState>>,
    Path(short_id): Path<String>,
) -> Response {
    match db::get_wine_image(&state.db, &short_id) {
        Ok(Some((bytes, mime))) => {
            let content_type = HeaderValue::from_str(&mime)
                .unwrap_or_else(|_| HeaderValue::from_static("application/octet-stream"));
            (
                StatusCode::OK,
                [(axum::http::header::CONTENT_TYPE, content_type)],
                bytes,
            )
                .into_response()
        }
        Ok(None) => StatusCode::NOT_FOUND.into_response(),
        Err(e) => {
            tracing::error!("Failed to get wine image: {}", e);
            StatusCode::INTERNAL_SERVER_ERROR.into_response()
        }
    }
}

fn xml_escape(s: &str) -> String {
    s.replace('&', "&amp;")
        .replace('<', "&lt;")
        .replace('>', "&gt;")
        .replace('"', "&quot;")
        .replace('\'', "&apos;")
}

fn to_rfc2822(sqlite_ts: &str) -> String {
    chrono::NaiveDateTime::parse_from_str(sqlite_ts, "%Y-%m-%d %H:%M:%S")
        .map(|naive| naive.and_utc().to_rfc2822())
        .unwrap_or_else(|_| sqlite_ts.to_string())
}

pub async fn rss_feed(State(state): State<Arc<AppState>>) -> Response {
    let site_url = &state.site_url;

    let wines = match db::get_cellar_wines(&state.db) {
        Ok(wines) => wines,
        Err(e) => {
            tracing::error!("Failed to get wines for RSS: {}", e);
            return (StatusCode::INTERNAL_SERVER_ERROR, "Server error").into_response();
        }
    };

    let mut items = String::new();
    for wine in &wines {
        let link = format!("{}/wines/{}", site_url, xml_escape(&wine.short_id));
        let title = xml_escape(&wine.name);
        let mut desc_parts: Vec<String> = Vec::new();
        if !wine.origin.is_empty() {
            desc_parts.push(format!("Origin: {}", wine.origin));
        }
        if !wine.grape.is_empty() {
            desc_parts.push(format!("Grape: {}", wine.grape));
        }
        if !wine.notes.is_empty() {
            desc_parts.push(wine.notes.clone());
        }
        let description = xml_escape(&desc_parts.join(" — "));
        let pub_date = to_rfc2822(&wine.created_at);
        let guid = format!("{}/wines/{}", site_url, xml_escape(&wine.short_id));

        items.push_str(&format!(
            "    <item>\n      <title>{title}</title>\n      <link>{link}</link>\n      <guid>{guid}</guid>\n      <description>{description}</description>\n      <pubDate>{pub_date}</pubDate>\n    </item>\n"
        ));
    }

    let last_build = wines
        .first()
        .map(|w| to_rfc2822(&w.created_at))
        .unwrap_or_default();

    let xml = format!(
        r#"<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:atom="http://www.w3.org/2005/Atom">
  <channel>
    <title>{title}</title>
    <link>{site_url}</link>
    <description>{desc}</description>
    <lastBuildDate>{last_build}</lastBuildDate>
    <atom:link href="{site_url}/feed.xml" rel="self" type="application/rss+xml"/>
{items}  </channel>
</rss>"#,
        title = xml_escape(&state.site_title),
        desc = xml_escape(&state.site_description),
        site_url = site_url,
        last_build = last_build,
        items = items,
    );

    (
        StatusCode::OK,
        [(
            axum::http::header::CONTENT_TYPE,
            HeaderValue::from_static("application/rss+xml; charset=utf-8"),
        )],
        xml,
    )
        .into_response()
}

pub async fn get_wishlist(
    State(state): State<Arc<AppState>>,
    headers: axum::http::HeaderMap,
) -> Response {
    let is_admin = auth::is_authenticated(&state, &headers);
    match db::get_wishlist_wines(&state.db) {
        Ok(wines) => WebTemplate(WishlistTemplate { wines, is_admin }).into_response(),
        Err(e) => {
            tracing::error!("Failed to list wishlist: {}", e);
            (StatusCode::INTERNAL_SERVER_ERROR, Html("Server error".to_string())).into_response()
        }
    }
}
