use std::sync::Arc;

use askama::Template;
use axum::{
    extract::{Path, State},
    http::{header, StatusCode},
    response::{Html, IntoResponse, Json, Response},
    routing::get,
    Router,
};
use chrono::Utc;
use rust_embed::Embed;
use serde::Serialize;

use crate::db::{self, DailyArtwork, Db};
use crate::scheduler;

#[derive(Embed)]
#[folder = "static/"]
struct Static;

pub struct AppState {
    pub db: Db,
    pub http: reqwest::Client,
    pub tz: chrono_tz::Tz,
    pub classifications: Vec<String>,
    pub exclude_terms: Vec<String>,
    pub backfill_days: u32,
    pub max_dedup_retries: u32,
}

#[derive(Template)]
#[template(path = "index.html")]
struct IndexTemplate {
    today_date: String,
    artwork: Option<ArtworkView>,
}

#[derive(Template)]
#[template(path = "day.html")]
struct DayTemplate {
    date: String,
    artwork: ArtworkView,
}

#[derive(Template)]
#[template(path = "archive.html")]
struct ArchiveTemplate {
    archive: Vec<ArchiveRow>,
}

#[derive(Template)]
#[template(path = "error.html")]
struct ErrorTemplate {
    title: String,
    message: String,
}

struct ArtworkView {
    date: String,
    title: String,
    artist_display: String,
    date_display: String,
    medium_display: String,
    dimensions: String,
    place_of_origin: String,
    credit_line: String,
    description: String,
    short_description: String,
    image_url: String,
    source_url: String,
}

struct ArchiveRow {
    date: String,
    title: String,
    artist: String,
}

fn iiif_url(image_id: &str) -> String {
    format!("https://www.artic.edu/iiif/2/{image_id}/full/843,/0/default.jpg")
}

fn source_url(artwork_id: i64) -> String {
    format!("https://www.artic.edu/artworks/{artwork_id}")
}

fn to_view(a: DailyArtwork) -> ArtworkView {
    ArtworkView {
        date: a.date,
        title: a.title,
        artist_display: a.artist_display.unwrap_or_default(),
        date_display: a.date_display.unwrap_or_default(),
        medium_display: a.medium_display.unwrap_or_default(),
        dimensions: a.dimensions.unwrap_or_default(),
        place_of_origin: a.place_of_origin.unwrap_or_default(),
        credit_line: a.credit_line.unwrap_or_default(),
        description: a.description.unwrap_or_default(),
        short_description: a.short_description.unwrap_or_default(),
        image_url: iiif_url(&a.image_id),
        source_url: source_url(a.artwork_id),
    }
}

fn to_archive_row(a: &DailyArtwork) -> ArchiveRow {
    ArchiveRow {
        date: a.date.clone(),
        title: a.title.clone(),
        artist: a
            .artist_title
            .clone()
            .or_else(|| a.artist_display.clone())
            .unwrap_or_default(),
    }
}

fn render<T: Template>(t: T) -> Response {
    match t.render() {
        Ok(body) => Html(body).into_response(),
        Err(e) => {
            tracing::error!("render failed: {e}");
            (StatusCode::INTERNAL_SERVER_ERROR, "render error").into_response()
        }
    }
}

async fn index_handler(State(state): State<Arc<AppState>>) -> Response {
    let today = scheduler::today_in_tz(&state.tz);
    let artwork = match db::get_daily(&state.db, &today) {
        Ok(Some(a)) => Some(to_view(a)),
        Ok(None) => None,
        Err(e) => {
            tracing::error!("index db error: {e}");
            return render(ErrorTemplate {
                title: "Error".to_string(),
                message: "Could not load today's artwork.".to_string(),
            });
        }
    };
    render(IndexTemplate {
        today_date: today,
        artwork,
    })
}

async fn day_handler(
    State(state): State<Arc<AppState>>,
    Path(date): Path<String>,
) -> Response {
    let parsed = match scheduler::parse_date(&date) {
        Some(d) => d,
        None => {
            return (
                StatusCode::BAD_REQUEST,
                render(ErrorTemplate {
                    title: "Invalid date".to_string(),
                    message: format!("'{date}' is not a valid YYYY-MM-DD date."),
                }),
            )
                .into_response();
        }
    };
    let today = scheduler::today_in_tz(&state.tz);
    if date.as_str() > today.as_str() {
        return (
            StatusCode::NOT_FOUND,
            render(ErrorTemplate {
                title: "Not yet".to_string(),
                message: format!(
                    "{} is in the future. The next day's artwork is not available until midnight {}.",
                    parsed, state.tz.name()
                ),
            }),
        )
            .into_response();
    }
    let artwork = match db::get_daily(&state.db, &date) {
        Ok(Some(a)) => to_view(a),
        Ok(None) => {
            return (
                StatusCode::NOT_FOUND,
                render(ErrorTemplate {
                    title: "Not found".to_string(),
                    message: format!("No artwork stored for {date}."),
                }),
            )
                .into_response();
        }
        Err(e) => {
            tracing::error!("day db error: {e}");
            return (
                StatusCode::INTERNAL_SERVER_ERROR,
                render(ErrorTemplate {
                    title: "Error".to_string(),
                    message: "Database error.".to_string(),
                }),
            )
                .into_response();
        }
    };
    render(DayTemplate {
        date,
        artwork,
    })
}

async fn archive_handler(State(state): State<Arc<AppState>>) -> Response {
    let archive = db::list_daily(&state.db, 1000)
        .unwrap_or_default()
        .iter()
        .map(to_archive_row)
        .collect();
    render(ArchiveTemplate { archive })
}

#[derive(Serialize)]
struct ApiArtwork<'a> {
    date: &'a str,
    artwork_id: i64,
    title: &'a str,
    artist_display: Option<&'a str>,
    date_display: Option<&'a str>,
    medium_display: Option<&'a str>,
    dimensions: Option<&'a str>,
    place_of_origin: Option<&'a str>,
    credit_line: Option<&'a str>,
    short_description: Option<&'a str>,
    image_id: &'a str,
    image_url: String,
    source_url: String,
}

fn to_api<'a>(a: &'a DailyArtwork) -> ApiArtwork<'a> {
    ApiArtwork {
        date: &a.date,
        artwork_id: a.artwork_id,
        title: &a.title,
        artist_display: a.artist_display.as_deref(),
        date_display: a.date_display.as_deref(),
        medium_display: a.medium_display.as_deref(),
        dimensions: a.dimensions.as_deref(),
        place_of_origin: a.place_of_origin.as_deref(),
        credit_line: a.credit_line.as_deref(),
        short_description: a.short_description.as_deref(),
        image_id: &a.image_id,
        image_url: iiif_url(&a.image_id),
        source_url: source_url(a.artwork_id),
    }
}

async fn api_today(State(state): State<Arc<AppState>>) -> Response {
    let today = scheduler::today_in_tz(&state.tz);
    match db::get_daily(&state.db, &today) {
        Ok(Some(a)) => Json(to_api(&a)).into_response(),
        Ok(None) => (
            StatusCode::NOT_FOUND,
            Json(serde_json::json!({"error": "today not yet populated"})),
        )
            .into_response(),
        Err(e) => {
            tracing::error!("api_today db error: {e}");
            StatusCode::INTERNAL_SERVER_ERROR.into_response()
        }
    }
}

async fn api_day(
    State(state): State<Arc<AppState>>,
    Path(date): Path<String>,
) -> Response {
    if scheduler::parse_date(&date).is_none() {
        return (
            StatusCode::BAD_REQUEST,
            Json(serde_json::json!({"error": "invalid date format"})),
        )
            .into_response();
    }
    let today = scheduler::today_in_tz(&state.tz);
    if date.as_str() > today.as_str() {
        return (
            StatusCode::NOT_FOUND,
            Json(serde_json::json!({"error": "future date"})),
        )
            .into_response();
    }
    match db::get_daily(&state.db, &date) {
        Ok(Some(a)) => Json(to_api(&a)).into_response(),
        Ok(None) => (
            StatusCode::NOT_FOUND,
            Json(serde_json::json!({"error": "no record for date"})),
        )
            .into_response(),
        Err(e) => {
            tracing::error!("api_day db error: {e}");
            StatusCode::INTERNAL_SERVER_ERROR.into_response()
        }
    }
}

async fn api_archive(State(state): State<Arc<AppState>>) -> Response {
    match db::list_daily(&state.db, 1000) {
        Ok(items) => {
            let out: Vec<ApiArtwork> = items.iter().map(to_api).collect();
            Json(out).into_response()
        }
        Err(e) => {
            tracing::error!("api_archive db error: {e}");
            StatusCode::INTERNAL_SERVER_ERROR.into_response()
        }
    }
}

async fn static_handler(Path(path): Path<String>) -> Response {
    match Static::get(&path) {
        Some(file) => {
            let mime = mime_guess::from_path(&path).first_or_octet_stream();
            (
                [(header::CONTENT_TYPE, mime.as_ref())],
                file.data.to_vec(),
            )
                .into_response()
        }
        None => StatusCode::NOT_FOUND.into_response(),
    }
}

pub async fn run() {
    let db_path = std::env::var("EASEL_DB_PATH").unwrap_or_else(|_| "easel.sqlite".to_string());
    let tz_name = std::env::var("EASEL_TIMEZONE").unwrap_or_else(|_| "UTC".to_string());
    let tz: chrono_tz::Tz = tz_name.parse().unwrap_or_else(|_| {
        tracing::warn!("invalid EASEL_TIMEZONE={tz_name}, falling back to UTC");
        chrono_tz::UTC
    });
    let classifications: Vec<String> = std::env::var("EASEL_CLASSIFICATIONS")
        .unwrap_or_else(|_| "painting".to_string())
        .split(',')
        .map(|s| s.trim().to_string())
        .filter(|s| !s.is_empty())
        .collect();
    if classifications.is_empty() {
        panic!("EASEL_CLASSIFICATIONS resolved to empty list");
    }
    let exclude_terms: Vec<String> = std::env::var("EASEL_EXCLUDE_TERMS")
        .unwrap_or_else(|_| "erotic,erotica,shunga".to_string())
        .split(',')
        .map(|s| s.trim().to_string())
        .filter(|s| !s.is_empty())
        .collect();
    let backfill_days: u32 = std::env::var("EASEL_BACKFILL_DAYS")
        .ok()
        .and_then(|v| v.parse().ok())
        .unwrap_or(0);
    let max_dedup_retries: u32 = std::env::var("EASEL_MAX_DEDUP_RETRIES")
        .ok()
        .and_then(|v| v.parse().ok())
        .unwrap_or(10);

    let db = db::init_db(&db_path);
    let http = crate::aic::build_client();

    let state = Arc::new(AppState {
        db,
        http,
        tz,
        classifications: classifications.clone(),
        exclude_terms: exclude_terms.clone(),
        backfill_days,
        max_dedup_retries,
    });

    tracing::info!(
        "easel starting: tz={} classifications={:?} exclude_terms={:?} backfill_days={} retries={}",
        state.tz.name(),
        classifications,
        exclude_terms,
        backfill_days,
        max_dedup_retries
    );
    tracing::info!("startup time: {}", Utc::now().to_rfc3339());

    tokio::spawn(scheduler::run(state.clone()));

    let app = Router::new()
        .route("/", get(index_handler))
        .route("/day/{date}", get(day_handler))
        .route("/archive", get(archive_handler))
        .route("/api/today", get(api_today))
        .route("/api/day/{date}", get(api_day))
        .route("/api/archive", get(api_archive))
        .route("/static/{*path}", get(static_handler))
        .merge(andromeda_darkmatter_css::router::<Arc<AppState>>())
        .with_state(state);

    let host = std::env::var("HOST").unwrap_or_else(|_| "127.0.0.1".to_string());
    let port: u16 = std::env::var("PORT")
        .ok()
        .and_then(|v| v.parse().ok())
        .unwrap_or(4242);
    let addr = format!("{host}:{port}");
    let listener = tokio::net::TcpListener::bind(&addr)
        .await
        .unwrap_or_else(|_| panic!("failed to bind {addr}"));
    tracing::info!("easel listening on http://{addr}");
    axum::serve(listener, app).await.expect("axum serve");
}
