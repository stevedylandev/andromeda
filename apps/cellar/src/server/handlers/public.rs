use askama_web::WebTemplate;
use axum::{
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
