use askama::Template;
use image::ImageDecoder;
use askama_web::WebTemplate;
use axum::{
    extract::{DefaultBodyLimit, Multipart, Path, Query, State},
    http::{HeaderValue, StatusCode},
    response::{Html, IntoResponse, Json, Redirect, Response},
    routing::{get, post},
    Router,
};
use rust_embed::Embed;
use std::sync::Arc;

use crate::auth;
use crate::claude;
use crate::db::{self, Db, Wine};

#[derive(Clone)]
pub struct AppState {
    pub db: Db,
    pub app_password: String,
    pub cookie_secure: bool,
    pub anthropic_api_key: Option<String>,
}

#[derive(Embed)]
#[folder = "static/"]
struct Static;

// --- Templates ---

#[derive(Template)]
#[template(path = "base.html")]
struct BaseTemplate;

#[derive(Template)]
#[template(path = "login.html")]
struct LoginTemplate {
    error: Option<String>,
    next: Option<String>,
}

struct WineWithSvg {
    wine: Wine,
    pentagon_svg: String,
}

#[derive(Template)]
#[template(path = "index.html")]
struct IndexTemplate {
    wines: Vec<WineWithSvg>,
}

#[derive(Template)]
#[template(path = "wine.html")]
struct WineDetailTemplate {
    wine: Wine,
    pentagon_svg: String,
}

#[derive(Template)]
#[template(path = "admin.html")]
struct AdminTemplate {
    wines: Vec<Wine>,
}

#[derive(Template)]
#[template(path = "wine_form.html")]
struct WineFormTemplate {
    wine: Option<Wine>,
    error: Option<String>,
    has_anthropic_key: bool,
}

// --- Query/Form structs ---

#[derive(serde::Deserialize, Default)]
pub struct FlashQuery {
    pub error: Option<String>,
    pub next: Option<String>,
}

#[derive(serde::Deserialize)]
struct LoginForm {
    password: String,
}

// --- Static file handlers ---

fn mime_from_path(path: &str) -> &'static str {
    match path.rsplit('.').next().unwrap_or("") {
        "css" => "text/css",
        "js" => "application/javascript",
        "html" => "text/html",
        "png" => "image/png",
        "jpg" | "jpeg" => "image/jpeg",
        "ico" => "image/x-icon",
        "svg" => "image/svg+xml",
        "woff" | "woff2" => "font/woff2",
        "ttf" => "font/ttf",
        "otf" => "font/otf",
        "json" | "webmanifest" => "application/json",
        _ => "application/octet-stream",
    }
}

async fn serve_static(Path(path): Path<String>) -> Response {
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

// --- Pentagon SVG ---

fn build_pentagon_svg(
    sweetness: i32,
    acidity: i32,
    tannin: i32,
    alcohol: i32,
    body: i32,
    size: f64,
    show_labels: bool,
) -> String {
    let cx = size / 2.0;
    let cy = size / 2.0;
    let margin = if show_labels { 30.0 } else { 5.0 };
    let r = size / 2.0 - margin;

    let scores = [sweetness, acidity, tannin, alcohol, body];
    let labels = ["Sweetness", "Acidity", "Tannin", "Alcohol", "Body"];

    let angles: Vec<f64> = (0..5)
        .map(|i| (-90.0_f64 + 72.0 * i as f64).to_radians())
        .collect();

    let mut svg = format!(
        r#"<svg viewBox="0 0 {s} {s}" width="{s}" height="{s}" xmlns="http://www.w3.org/2000/svg">"#,
        s = size
    );

    // Grid pentagons at 20%, 40%, 60%, 80%
    for pct in &[0.2, 0.4, 0.6, 0.8] {
        let points: String = angles
            .iter()
            .map(|a| format!("{:.1},{:.1}", cx + r * pct * a.cos(), cy + r * pct * a.sin()))
            .collect::<Vec<_>>()
            .join(" ");
        svg.push_str(&format!(
            r#"<polygon points="{}" fill="none" stroke="white" stroke-opacity="0.12" stroke-width="0.75"/>"#,
            points
        ));
    }

    // Outer pentagon (100%)
    let outline: String = angles
        .iter()
        .map(|a| format!("{:.1},{:.1}", cx + r * a.cos(), cy + r * a.sin()))
        .collect::<Vec<_>>()
        .join(" ");
    svg.push_str(&format!(
        r#"<polygon points="{}" fill="none" stroke="white" stroke-opacity="0.25" stroke-width="1"/>"#,
        outline
    ));

    // Axis lines from center to each vertex
    for a in &angles {
        svg.push_str(&format!(
            r#"<line x1="{:.1}" y1="{:.1}" x2="{:.1}" y2="{:.1}" stroke="white" stroke-opacity="0.12" stroke-width="0.75"/>"#,
            cx, cy, cx + r * a.cos(), cy + r * a.sin()
        ));
    }

    // Data polygon
    let data_points: Vec<(f64, f64)> = scores
        .iter()
        .zip(&angles)
        .map(|(s, a)| {
            let d = (*s as f64 / 5.0) * r;
            (cx + d * a.cos(), cy + d * a.sin())
        })
        .collect();

    let data_str: String = data_points
        .iter()
        .map(|(x, y)| format!("{:.1},{:.1}", x, y))
        .collect::<Vec<_>>()
        .join(" ");
    svg.push_str(&format!(
        r#"<polygon points="{}" fill="white" fill-opacity="0.08" stroke="white" stroke-width="1.5"/>"#,
        data_str
    ));

    // Data dots
    for (x, y) in &data_points {
        svg.push_str(&format!(
            r#"<circle cx="{:.1}" cy="{:.1}" r="2.5" fill="white"/>"#,
            x, y
        ));
    }

    // Labels
    if show_labels {
        for (i, label) in labels.iter().enumerate() {
            let a = angles[i];
            let label_dist = r + 18.0;
            let lx = cx + label_dist * a.cos();
            let ly = cy + label_dist * a.sin() + 3.5;
            svg.push_str(&format!(
                r#"<text x="{:.1}" y="{:.1}" fill="white" fill-opacity="0.5" font-size="9" font-family="Commit Mono, monospace" text-anchor="middle">{}</text>"#,
                lx, ly, label
            ));
        }
    }

    svg.push_str("</svg>");
    svg
}

// --- Auth handlers ---

async fn get_login(Query(q): Query<FlashQuery>) -> Response {
    WebTemplate(LoginTemplate { error: q.error, next: q.next }).into_response()
}

async fn post_login(
    Query(q): Query<FlashQuery>,
    State(state): State<Arc<AppState>>,
    axum::extract::Form(form): axum::extract::Form<LoginForm>,
) -> Response {
    let next = q.next.as_deref().unwrap_or("/admin");
    if !auth::verify_password(&form.password, &state.app_password) {
        return Redirect::to(&format!("/admin/login?error=Invalid+password&next={}", urlencoded(next))).into_response();
    }

    let token = auth::generate_session_token();

    let expires_at = {
        use std::time::{SystemTime, UNIX_EPOCH};
        let secs = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap()
            .as_secs()
            + 7 * 24 * 3600;
        let days = secs / 86400;
        let tod = secs % 86400;
        let (y, m, d) = days_to_ymd(days as i64);
        format!(
            "{:04}-{:02}-{:02} {:02}:{:02}:{:02}",
            y,
            m,
            d,
            tod / 3600,
            (tod % 3600) / 60,
            tod % 60
        )
    };

    if let Err(e) = db::insert_session(&state.db, &token, &expires_at) {
        tracing::error!("Failed to create session: {}", e);
        return Redirect::to("/admin/login?error=Server+error").into_response();
    }

    let _ = db::prune_expired_sessions(&state.db);

    let cookie = auth::build_session_cookie(&token, state.cookie_secure);
    // Only allow relative redirects to prevent open redirect
    let redirect_to = if next.starts_with('/') { next } else { "/admin" };
    let mut resp = Redirect::to(redirect_to).into_response();
    resp.headers_mut().insert(
        axum::http::header::SET_COOKIE,
        HeaderValue::from_str(&cookie).unwrap(),
    );
    resp
}

async fn get_logout(State(state): State<Arc<AppState>>, headers: axum::http::HeaderMap) -> Response {
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

// --- Public handlers ---

async fn get_index(State(state): State<Arc<AppState>>) -> Response {
    match db::get_all_wines(&state.db) {
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

async fn get_wine_detail(
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
            WebTemplate(WineDetailTemplate { wine, pentagon_svg }).into_response()
        }
        Ok(None) => (StatusCode::NOT_FOUND, Html("Wine not found".to_string())).into_response(),
        Err(e) => {
            tracing::error!("Failed to get wine: {}", e);
            (StatusCode::INTERNAL_SERVER_ERROR, Html("Server error".to_string())).into_response()
        }
    }
}

async fn get_wine_image(
    State(state): State<Arc<AppState>>,
    Path(short_id): Path<String>,
) -> Response {
    match db::get_wine_image(&state.db, &short_id) {
        Ok(Some((bytes, mime))) => {
            let content_type = HeaderValue::from_str(&mime).unwrap_or_else(|_| {
                HeaderValue::from_static("application/octet-stream")
            });
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

// --- Admin handlers ---

async fn get_admin(
    _session: auth::AuthSession,
    State(state): State<Arc<AppState>>,
) -> Response {
    match db::get_all_wines(&state.db) {
        Ok(wines) => WebTemplate(AdminTemplate { wines }).into_response(),
        Err(e) => {
            tracing::error!("Failed to list wines: {}", e);
            (StatusCode::INTERNAL_SERVER_ERROR, Html("Server error".to_string())).into_response()
        }
    }
}

async fn get_new_wine(
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

async fn get_edit_wine(
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

// --- Image processing ---

fn process_image(data: &[u8]) -> Result<Vec<u8>, String> {
    let reader = image::ImageReader::new(std::io::Cursor::new(data))
        .with_guessed_format()
        .map_err(|e| format!("Failed to read image: {}", e))?;
    let mut decoder = reader
        .into_decoder()
        .map_err(|e| format!("Failed to create decoder: {}", e))?;
    let orientation = decoder.orientation().unwrap_or(image::metadata::Orientation::NoTransforms);
    let mut img = image::DynamicImage::from_decoder(decoder)
        .map_err(|e| format!("Failed to decode image: {}", e))?;
    img.apply_orientation(orientation);
    let mut output = Vec::new();
    let encoder = image::codecs::jpeg::JpegEncoder::new_with_quality(&mut output, 75);
    img.write_with_encoder(encoder)
        .map_err(|e| format!("JPEG encoding failed: {}", e))?;
    Ok(output)
}

// --- Multipart parsing ---

struct WineFormData {
    name: String,
    origin: String,
    grape: String,
    notes: String,
    background: String,
    image: Option<Vec<u8>>,
    image_mime: Option<String>,
    sweetness: i32,
    acidity: i32,
    tannin: i32,
    alcohol: i32,
    body: i32,
}

async fn parse_wine_multipart(mut multipart: Multipart) -> Result<WineFormData, String> {
    let mut name = String::new();
    let mut origin = String::new();
    let mut grape = String::new();
    let mut notes = String::new();
    let mut background = String::new();
    let mut image: Option<Vec<u8>> = None;
    let mut image_mime: Option<String> = None;
    let mut sweetness = 3;
    let mut acidity = 3;
    let mut tannin = 3;
    let mut alcohol = 3;
    let mut body = 3;

    while let Ok(Some(field)) = multipart.next_field().await {
        let field_name = field.name().unwrap_or("").to_string();
        match field_name.as_str() {
            "image" => {
                let bytes = field.bytes().await.map_err(|e| format!("Failed to read image: {}", e))?;
                if !bytes.is_empty() {
                    let processed = process_image(&bytes)?;
                    image = Some(processed);
                    image_mime = Some("image/jpeg".to_string());
                }
            }
            "name" => name = field.text().await.unwrap_or_default(),
            "origin" => origin = field.text().await.unwrap_or_default(),
            "grape" => grape = field.text().await.unwrap_or_default(),
            "notes" => notes = field.text().await.unwrap_or_default(),
            "background" => background = field.text().await.unwrap_or_default(),
            "sweetness" => sweetness = field.text().await.unwrap_or_default().parse().unwrap_or(3),
            "acidity" => acidity = field.text().await.unwrap_or_default().parse().unwrap_or(3),
            "tannin" => tannin = field.text().await.unwrap_or_default().parse().unwrap_or(3),
            "alcohol" => alcohol = field.text().await.unwrap_or_default().parse().unwrap_or(3),
            "body" => body = field.text().await.unwrap_or_default().parse().unwrap_or(3),
            _ => {}
        }
    }

    if name.trim().is_empty() {
        return Err("Name is required".to_string());
    }

    // Clamp scores to 1-5
    let clamp = |v: i32| v.max(1).min(5);
    Ok(WineFormData {
        name: name.trim().to_string(),
        origin: origin.trim().to_string(),
        grape: grape.trim().to_string(),
        notes: notes.trim().to_string(),
        background: background.trim().to_string(),
        image,
        image_mime,
        sweetness: clamp(sweetness),
        acidity: clamp(acidity),
        tannin: clamp(tannin),
        alcohol: clamp(alcohol),
        body: clamp(body),
    })
}

async fn post_new_wine(
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

    match db::create_wine(
        &state.db,
        &data.name,
        &data.origin,
        &data.grape,
        &data.notes,
        data.image.as_deref(),
        data.image_mime.as_deref(),
        data.sweetness,
        data.acidity,
        data.tannin,
        data.alcohol,
        data.body,
        &data.background,
    ) {
        Ok(wine) => Redirect::to(&format!("/wines/{}", wine.short_id)).into_response(),
        Err(e) => {
            tracing::error!("Failed to create wine: {}", e);
            Redirect::to("/admin/new?error=Failed+to+create+wine").into_response()
        }
    }
}

async fn post_edit_wine(
    _session: auth::AuthSession,
    State(state): State<Arc<AppState>>,
    Path(short_id): Path<String>,
    multipart: Multipart,
) -> Response {
    let data = match parse_wine_multipart(multipart).await {
        Ok(data) => data,
        Err(e) => {
            return Redirect::to(&format!("/admin/edit/{}?error={}", short_id, urlencoded(&e)))
                .into_response();
        }
    };

    match db::update_wine(
        &state.db,
        &short_id,
        &data.name,
        &data.origin,
        &data.grape,
        &data.notes,
        data.sweetness,
        data.acidity,
        data.tannin,
        data.alcohol,
        data.body,
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

async fn post_delete_wine(
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

// --- Claude vision handler ---

async fn post_analyze_image(
    _session: auth::AuthSession,
    State(state): State<Arc<AppState>>,
    mut multipart: Multipart,
) -> Response {
    let api_key = match &state.anthropic_api_key {
        Some(key) => key.clone(),
        None => {
            return (StatusCode::BAD_REQUEST, Json(serde_json::json!({"error": "No API key configured"})))
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
            return (StatusCode::BAD_REQUEST, Json(serde_json::json!({"error": "No image provided"})))
                .into_response();
        }
    };

    match claude::analyze_wine_image(&api_key, &image_bytes, &media_type).await {
        Ok(result) => (StatusCode::OK, Json(result)).into_response(),
        Err(e) => {
            tracing::error!("Claude analysis failed: {}", e);
            (StatusCode::INTERNAL_SERVER_ERROR, Json(serde_json::json!({"error": e})))
                .into_response()
        }
    }
}

// --- Helpers ---

fn urlencoded(s: &str) -> String {
    s.replace(' ', "+")
        .replace('&', "%26")
        .replace('=', "%3D")
}

fn days_to_ymd(mut days: i64) -> (i64, i64, i64) {
    days += 719468;
    let era = if days >= 0 { days } else { days - 146096 } / 146097;
    let doe = (days - era * 146097) as u32;
    let yoe = (doe - doe / 1460 + doe / 36524 - doe / 146096) / 365;
    let y = yoe as i64 + era * 400;
    let doy = doe - (365 * yoe + yoe / 4 - yoe / 100);
    let mp = (5 * doy + 2) / 153;
    let d = doy - (153 * mp + 2) / 5 + 1;
    let m = if mp < 10 { mp + 3 } else { mp - 9 };
    let y = if m <= 2 { y + 1 } else { y };
    (y, m as i64, d as i64)
}

// --- Router ---

pub async fn run(host: String, port: u16) {
    dotenvy::dotenv().ok();

    let db = db::init_db();

    if let Err(e) = db::prune_expired_sessions(&db) {
        tracing::warn!("Failed to prune sessions: {}", e);
    }

    let app_password = std::env::var("CELLAR_PASSWORD").unwrap_or_else(|_| {
        tracing::warn!("CELLAR_PASSWORD not set, using default 'changeme'");
        "changeme".to_string()
    });

    let cookie_secure = std::env::var("COOKIE_SECURE")
        .map(|v| v == "true")
        .unwrap_or(false);

    let anthropic_api_key = std::env::var("ANTHROPIC_API_KEY").ok().filter(|k| !k.is_empty());

    let state = Arc::new(AppState {
        db,
        app_password,
        cookie_secure,
        anthropic_api_key,
    });

    let app = Router::new()
        // Public routes
        .route("/", get(get_index))
        .route("/wines/{short_id}", get(get_wine_detail))
        .route("/wines/{short_id}/image", get(get_wine_image))
        // Admin auth routes
        .route("/admin/login", get(get_login).post(post_login))
        .route("/admin/logout", get(get_logout))
        // Admin protected routes
        .route("/admin", get(get_admin))
        .route("/admin/new", get(get_new_wine).post(post_new_wine))
        .route("/admin/edit/{short_id}", get(get_edit_wine).post(post_edit_wine))
        .route("/admin/delete/{short_id}", post(post_delete_wine))
        // Claude vision
        .route("/admin/analyze-image", post(post_analyze_image))
        // Static assets
        .route("/static/{*path}", get(serve_static))
        .layer(DefaultBodyLimit::max(10 * 1024 * 1024))
        .with_state(state);

    let addr = format!("{}:{}", host, port);
    tracing::info!("Listening on http://{}", addr);

    let listener = tokio::net::TcpListener::bind(&addr).await.unwrap();
    axum::serve(listener, app).await.unwrap();
}
