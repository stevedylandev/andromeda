use askama_web::WebTemplate;
use axum::{
    extract::{Multipart, Path, Query, State},
    http::{HeaderValue, StatusCode},
    response::{Html, IntoResponse, Json, Redirect, Response},
};
use std::sync::Arc;

use super::super::*;
use crate::{auth, claude, db};

// --- Auth handlers ---

pub async fn get_login(Query(q): Query<FlashQuery>) -> Response {
    WebTemplate(LoginTemplate { error: q.error, next: q.next }).into_response()
}

pub async fn post_login(
    Query(q): Query<FlashQuery>,
    State(state): State<Arc<AppState>>,
    axum::extract::Form(form): axum::extract::Form<LoginForm>,
) -> Response {
    let next = q.next.as_deref().unwrap_or("/admin");
    if !auth::verify_password(&form.password, &state.app_password) {
        return Redirect::to(&format!(
            "/admin/login?error=Invalid+password&next={}",
            urlencoded(next)
        ))
        .into_response();
    }

    let token = auth::generate_session_token();

    let expires_at = andromeda_auth::datetime::expiry_datetime_string(7 * 24 * 3600);

    if let Err(e) = db::insert_session(&state.db, &token, &expires_at) {
        tracing::error!("Failed to create session: {}", e);
        return Redirect::to("/admin/login?error=Server+error").into_response();
    }

    let _ = db::prune_expired_sessions(&state.db);

    let cookie = auth::build_session_cookie(&token, state.cookie_secure);
    let redirect_to = if next.starts_with('/') { next } else { "/admin" };
    let mut resp = Redirect::to(redirect_to).into_response();
    resp.headers_mut().insert(
        axum::http::header::SET_COOKIE,
        HeaderValue::from_str(&cookie).unwrap(),
    );
    resp
}

pub async fn get_logout(
    State(state): State<Arc<AppState>>,
    headers: axum::http::HeaderMap,
) -> Response {
    if let Some(cookie_header) = headers.get("cookie").and_then(|v| v.to_str().ok()) {
        for part in cookie_header.split(';') {
            let part = part.trim();
            if let Some(val) = part.strip_prefix("session=") {
                let val = val.trim();
                if !val.is_empty() {
                    let _ = db::delete_session(&state.db, val);
                }
            }
        }
    }

    let cookie = auth::clear_session_cookie();
    let mut resp = Redirect::to("/admin/login").into_response();
    resp.headers_mut().insert(
        axum::http::header::SET_COOKIE,
        HeaderValue::from_str(&cookie).unwrap(),
    );
    resp
}

// --- Admin wine handlers ---

pub async fn get_admin(
    _session: auth::AuthSession,
    State(state): State<Arc<AppState>>,
) -> Response {
    match db::get_cellar_wines(&state.db) {
        Ok(wines) => WebTemplate(AdminTemplate { wines }).into_response(),
        Err(e) => {
            tracing::error!("Failed to list wines: {}", e);
            (StatusCode::INTERNAL_SERVER_ERROR, Html("Server error".to_string())).into_response()
        }
    }
}

pub async fn get_new_wine(
    _session: auth::AuthSession,
    State(state): State<Arc<AppState>>,
    Query(q): Query<FlashQuery>,
) -> Response {
    WebTemplate(WineFormTemplate {
        wine: None,
        error: q.error,
        has_anthropic_key: state.anthropic_api_key.is_some(),
    })
    .into_response()
}

pub async fn get_edit_wine(
    _session: auth::AuthSession,
    State(state): State<Arc<AppState>>,
    Path(short_id): Path<String>,
    Query(q): Query<FlashQuery>,
) -> Response {
    match db::get_wine_by_short_id(&state.db, &short_id) {
        Ok(Some(wine)) => WebTemplate(WineFormTemplate {
            wine: Some(wine),
            error: q.error,
            has_anthropic_key: state.anthropic_api_key.is_some(),
        })
        .into_response(),
        Ok(None) => (StatusCode::NOT_FOUND, Html("Wine not found".to_string())).into_response(),
        Err(e) => {
            tracing::error!("Failed to get wine: {}", e);
            (StatusCode::INTERNAL_SERVER_ERROR, Html("Server error".to_string())).into_response()
        }
    }
}

pub async fn post_new_wine(
    _session: auth::AuthSession,
    State(state): State<Arc<AppState>>,
    multipart: Multipart,
) -> Response {
    let data = match parse_wine_multipart(multipart).await {
        Ok(data) => data,
        Err(e) => {
            return Redirect::to(&format!("/admin/new?error={}", urlencoded(&e))).into_response();
        }
    };

    let input = db::WineInput {
        name: &data.name,
        origin: &data.origin,
        grape: &data.grape,
        notes: &data.notes,
        sweetness: data.sweetness,
        acidity: data.acidity,
        tannin: data.tannin,
        alcohol: data.alcohol,
        body: data.body,
        clarity: data.clarity,
        color_intensity: data.color_intensity,
        aroma_intensity: data.aroma_intensity,
        nose_complexity: data.nose_complexity,
        background: &data.background,
    };
    match db::create_wine(&state.db, &input, false) {
        Ok(wine) => {
            if let (Some(image), Some(mime)) = (&data.image, &data.image_mime) {
                if let Err(e) = db::update_wine_image(&state.db, &wine.short_id, image, mime) {
                    tracing::error!("Failed to set wine image: {}", e);
                }
            }
            Redirect::to(&format!("/wines/{}", wine.short_id)).into_response()
        }
        Err(e) => {
            tracing::error!("Failed to create wine: {}", e);
            Redirect::to("/admin/new?error=Failed+to+create+wine").into_response()
        }
    }
}

pub async fn post_edit_wine(
    _session: auth::AuthSession,
    State(state): State<Arc<AppState>>,
    Path(short_id): Path<String>,
    multipart: Multipart,
) -> Response {
    let data = match parse_wine_multipart(multipart).await {
        Ok(data) => data,
        Err(e) => {
            return Redirect::to(&format!(
                "/admin/edit/{}?error={}",
                short_id,
                urlencoded(&e)
            ))
            .into_response();
        }
    };

    let input = db::WineInput {
        name: &data.name,
        origin: &data.origin,
        grape: &data.grape,
        notes: &data.notes,
        sweetness: data.sweetness,
        acidity: data.acidity,
        tannin: data.tannin,
        alcohol: data.alcohol,
        body: data.body,
        clarity: data.clarity,
        color_intensity: data.color_intensity,
        aroma_intensity: data.aroma_intensity,
        nose_complexity: data.nose_complexity,
        background: &data.background,
    };
    match db::update_wine(&state.db, &short_id, &input) {
        Ok(Some(_)) => {
            if let Some(image) = &data.image {
                if let Some(mime) = &data.image_mime {
                    if let Err(e) = db::update_wine_image(&state.db, &short_id, image, mime) {
                        tracing::error!("Failed to update wine image: {}", e);
                    }
                }
            }
            Redirect::to(&format!("/wines/{}", short_id)).into_response()
        }
        Ok(None) => (StatusCode::NOT_FOUND, Html("Wine not found".to_string())).into_response(),
        Err(e) => {
            tracing::error!("Failed to update wine: {}", e);
            Redirect::to(&format!(
                "/admin/edit/{}?error=Failed+to+update+wine",
                short_id
            ))
            .into_response()
        }
    }
}

pub async fn post_delete_wine(
    _session: auth::AuthSession,
    State(state): State<Arc<AppState>>,
    Path(short_id): Path<String>,
) -> Response {
    match db::delete_wine(&state.db, &short_id) {
        Ok(_) => Redirect::to("/admin").into_response(),
        Err(e) => {
            tracing::error!("Failed to delete wine: {}", e);
            Redirect::to("/admin").into_response()
        }
    }
}

// --- Wishlist handlers ---

pub async fn get_new_wishlist_wine(
    _session: auth::AuthSession,
    State(state): State<Arc<AppState>>,
    Query(q): Query<FlashQuery>,
) -> Response {
    WebTemplate(WishlistFormTemplate {
        wine: None,
        error: q.error,
        has_anthropic_key: state.anthropic_api_key.is_some(),
    })
    .into_response()
}

pub async fn get_edit_wishlist_wine(
    _session: auth::AuthSession,
    State(state): State<Arc<AppState>>,
    Path(short_id): Path<String>,
    Query(q): Query<FlashQuery>,
) -> Response {
    match db::get_wine_by_short_id(&state.db, &short_id) {
        Ok(Some(wine)) => WebTemplate(WishlistFormTemplate {
            wine: Some(wine),
            error: q.error,
            has_anthropic_key: state.anthropic_api_key.is_some(),
        })
        .into_response(),
        Ok(None) => (StatusCode::NOT_FOUND, Html("Wine not found".to_string())).into_response(),
        Err(e) => {
            tracing::error!("Failed to get wine: {}", e);
            (StatusCode::INTERNAL_SERVER_ERROR, Html("Server error".to_string())).into_response()
        }
    }
}

pub async fn post_new_wishlist_wine(
    _session: auth::AuthSession,
    State(state): State<Arc<AppState>>,
    multipart: Multipart,
) -> Response {
    let data = match parse_wishlist_multipart(multipart).await {
        Ok(data) => data,
        Err(e) => {
            return Redirect::to(&format!("/admin/wishlist/new?error={}", urlencoded(&e)))
                .into_response();
        }
    };

    let input = db::WineInput {
        name: &data.name,
        origin: &data.origin,
        grape: &data.grape,
        notes: &data.notes,
        sweetness: 3,
        acidity: 3,
        tannin: 3,
        alcohol: 3,
        body: 3,
        clarity: 3,
        color_intensity: 3,
        aroma_intensity: 3,
        nose_complexity: 3,
        background: &data.background,
    };
    match db::create_wine(&state.db, &input, true) {
        Ok(wine) => {
            if let (Some(image), Some(mime)) = (&data.image, &data.image_mime) {
                if let Err(e) = db::update_wine_image(&state.db, &wine.short_id, image, mime) {
                    tracing::error!("Failed to set wine image: {}", e);
                }
            }
            Redirect::to("/wishlist").into_response()
        }
        Err(e) => {
            tracing::error!("Failed to create wishlist wine: {}", e);
            Redirect::to("/admin/wishlist/new?error=Failed+to+create+wine").into_response()
        }
    }
}

pub async fn post_edit_wishlist_wine(
    _session: auth::AuthSession,
    State(state): State<Arc<AppState>>,
    Path(short_id): Path<String>,
    multipart: Multipart,
) -> Response {
    let data = match parse_wishlist_multipart(multipart).await {
        Ok(data) => data,
        Err(e) => {
            return Redirect::to(&format!(
                "/admin/wishlist/edit/{}?error={}",
                short_id,
                urlencoded(&e)
            ))
            .into_response();
        }
    };

    match db::update_wishlist_wine(
        &state.db,
        &short_id,
        &data.name,
        &data.origin,
        &data.grape,
        &data.notes,
        &data.background,
    ) {
        Ok(Some(_)) => {
            if let Some(image) = &data.image {
                if let Some(mime) = &data.image_mime {
                    if let Err(e) = db::update_wine_image(&state.db, &short_id, image, mime) {
                        tracing::error!("Failed to update wine image: {}", e);
                    }
                }
            }
            Redirect::to("/wishlist").into_response()
        }
        Ok(None) => (StatusCode::NOT_FOUND, Html("Wine not found".to_string())).into_response(),
        Err(e) => {
            tracing::error!("Failed to update wishlist wine: {}", e);
            Redirect::to(&format!(
                "/admin/wishlist/edit/{}?error=Failed+to+update+wine",
                short_id
            ))
            .into_response()
        }
    }
}

pub async fn post_delete_wishlist_wine(
    _session: auth::AuthSession,
    State(state): State<Arc<AppState>>,
    Path(short_id): Path<String>,
) -> Response {
    match db::delete_wine(&state.db, &short_id) {
        Ok(_) => Redirect::to("/wishlist").into_response(),
        Err(e) => {
            tracing::error!("Failed to delete wine: {}", e);
            Redirect::to("/wishlist").into_response()
        }
    }
}

pub async fn post_promote_wine(
    _session: auth::AuthSession,
    State(state): State<Arc<AppState>>,
    Path(short_id): Path<String>,
) -> Response {
    match db::promote_wine(&state.db, &short_id) {
        Ok(true) => Redirect::to(&format!("/admin/edit/{}", short_id)).into_response(),
        Ok(false) => (StatusCode::NOT_FOUND, Html("Wine not found".to_string())).into_response(),
        Err(e) => {
            tracing::error!("Failed to promote wine: {}", e);
            Redirect::to("/wishlist").into_response()
        }
    }
}

// --- Claude vision handler ---

pub async fn post_analyze_image(
    _session: auth::AuthSession,
    State(state): State<Arc<AppState>>,
    mut multipart: Multipart,
) -> Response {
    let api_key = match &state.anthropic_api_key {
        Some(key) => key.clone(),
        None => {
            return (
                StatusCode::BAD_REQUEST,
                Json(serde_json::json!({"error": "No API key configured"})),
            )
                .into_response();
        }
    };

    let mut image_bytes: Option<Vec<u8>> = None;
    let mut media_type = String::from("image/jpeg");

    while let Ok(Some(field)) = multipart.next_field().await {
        if field.name() == Some("image") {
            media_type = field.content_type().unwrap_or("image/jpeg").to_string();
            if let Ok(bytes) = field.bytes().await {
                if !bytes.is_empty() {
                    image_bytes = Some(bytes.to_vec());
                }
            }
        }
    }

    let image_bytes = match image_bytes {
        Some(bytes) => bytes,
        None => {
            return (
                StatusCode::BAD_REQUEST,
                Json(serde_json::json!({"error": "No image provided"})),
            )
                .into_response();
        }
    };

    match claude::analyze_wine_image(&api_key, &image_bytes, &media_type).await {
        Ok(result) => (StatusCode::OK, Json(result)).into_response(),
        Err(e) => {
            tracing::error!("Claude analysis failed: {}", e);
            (
                StatusCode::INTERNAL_SERVER_ERROR,
                Json(serde_json::json!({"error": e})),
            )
                .into_response()
        }
    }
}
