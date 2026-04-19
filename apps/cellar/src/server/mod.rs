use askama::Template;
use axum::{
    extract::{multipart::Field, DefaultBodyLimit, Multipart},
    routing::{get, post},
    Router,
};
use image::ImageDecoder;
use rust_embed::Embed;
use std::sync::Arc;

use crate::db::{self, Db, Wine};

mod handlers;

#[derive(Clone)]
pub struct AppState {
    pub db: Db,
    pub app_password: String,
    pub cookie_secure: bool,
    pub anthropic_api_key: Option<String>,
    pub site_url: String,
    pub site_title: String,
    pub site_description: String,
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
    bars_svg: String,
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

#[derive(Template)]
#[template(path = "wishlist.html")]
struct WishlistTemplate {
    wines: Vec<Wine>,
    is_admin: bool,
}

#[derive(Template)]
#[template(path = "wishlist_form.html")]
struct WishlistFormTemplate {
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

// --- Helpers ---

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

fn urlencoded(s: &str) -> String {
    s.replace(' ', "+")
        .replace('&', "%26")
        .replace('=', "%3D")
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
        r#"<svg viewBox="0 0 {s} {s}" width="100%" xmlns="http://www.w3.org/2000/svg">"#,
        s = size
    );

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

    let outline: String = angles
        .iter()
        .map(|a| format!("{:.1},{:.1}", cx + r * a.cos(), cy + r * a.sin()))
        .collect::<Vec<_>>()
        .join(" ");
    svg.push_str(&format!(
        r#"<polygon points="{}" fill="none" stroke="white" stroke-opacity="0.25" stroke-width="1"/>"#,
        outline
    ));

    for a in &angles {
        svg.push_str(&format!(
            r#"<line x1="{:.1}" y1="{:.1}" x2="{:.1}" y2="{:.1}" stroke="white" stroke-opacity="0.12" stroke-width="0.75"/>"#,
            cx, cy, cx + r * a.cos(), cy + r * a.sin()
        ));
    }

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

    for (x, y) in &data_points {
        svg.push_str(&format!(
            r#"<circle cx="{:.1}" cy="{:.1}" r="2.5" fill="white"/>"#,
            x, y
        ));
    }

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

fn build_bars_svg(
    clarity: i32,
    color_intensity: i32,
    aroma_intensity: i32,
    nose_complexity: i32,
    width: f64,
) -> String {
    let bar_height = 4.0;
    let row_height = 22.0;
    let section_gap = 14.0;
    let label_width = 100.0;
    let track_left = label_width + 4.0;
    let track_width = width - track_left - 10.0;
    let header_size = 9.0;

    let sections: &[(&str, &[(&str, i32)])] = &[
        ("Appearance", &[("Clarity", clarity), ("Intensity", color_intensity)]),
        ("Nose", &[("Aroma", aroma_intensity), ("Complexity", nose_complexity)]),
    ];

    let total_rows: usize = sections.iter().map(|(_, attrs)| attrs.len()).sum();
    let total_height = (sections.len() as f64) * (header_size + 8.0)
        + (total_rows as f64) * row_height
        + section_gap;

    let mut svg = format!(
        r#"<svg viewBox="0 0 {w} {h}" width="100%" xmlns="http://www.w3.org/2000/svg">"#,
        w = width,
        h = total_height
    );

    let mut y = 4.0;

    for (si, (section_name, attrs)) in sections.iter().enumerate() {
        if si > 0 {
            y += section_gap;
        }

        svg.push_str(&format!(
            r#"<text x="0" y="{:.1}" fill="white" fill-opacity="0.4" font-size="{}" font-family="Commit Mono, monospace" text-transform="uppercase" letter-spacing="1">{}</text>"#,
            y + header_size, header_size, section_name
        ));
        y += header_size + 8.0;

        for (label, score) in *attrs {
            let bar_y = y + (row_height - bar_height) / 2.0;
            let fill_width = (*score as f64 / 5.0) * track_width;

            svg.push_str(&format!(
                r#"<text x="0" y="{:.1}" fill="white" fill-opacity="0.5" font-size="9" font-family="Commit Mono, monospace">{}</text>"#,
                y + row_height / 2.0 + 3.0, label
            ));

            svg.push_str(&format!(
                r#"<rect x="{:.1}" y="{:.1}" width="{:.1}" height="{:.1}" rx="2" fill="white" fill-opacity="0.08"/>"#,
                track_left, bar_y, track_width, bar_height
            ));

            if fill_width > 0.0 {
                svg.push_str(&format!(
                    r#"<rect x="{:.1}" y="{:.1}" width="{:.1}" height="{:.1}" rx="2" fill="white" fill-opacity="0.6"/>"#,
                    track_left, bar_y, fill_width, bar_height
                ));
            }

            y += row_height;
        }
    }

    svg.push_str("</svg>");
    svg
}

// --- Image processing ---

fn process_image(data: &[u8]) -> Result<Vec<u8>, String> {
    let reader = image::ImageReader::new(std::io::Cursor::new(data))
        .with_guessed_format()
        .map_err(|e| format!("Failed to read image: {}", e))?;
    let mut decoder = reader
        .into_decoder()
        .map_err(|e| format!("Failed to create decoder: {}", e))?;
    let orientation = decoder
        .orientation()
        .unwrap_or(image::metadata::Orientation::NoTransforms);
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

#[derive(Default)]
struct WineBase {
    name: String,
    origin: String,
    grape: String,
    notes: String,
    background: String,
    image: Option<Vec<u8>>,
    image_mime: Option<String>,
}

impl WineBase {
    fn owns(field_name: &str) -> bool {
        matches!(
            field_name,
            "image" | "name" | "origin" | "grape" | "notes" | "background"
        )
    }

    async fn apply_field(&mut self, field_name: &str, field: Field<'_>) -> Result<(), String> {
        match field_name {
            "image" => {
                let bytes = field.bytes().await.map_err(|e| format!("Failed to read image: {}", e))?;
                if !bytes.is_empty() {
                    self.image = Some(process_image(&bytes)?);
                    self.image_mime = Some("image/jpeg".to_string());
                }
            }
            "name" => self.name = field.text().await.unwrap_or_default(),
            "origin" => self.origin = field.text().await.unwrap_or_default(),
            "grape" => self.grape = field.text().await.unwrap_or_default(),
            "notes" => self.notes = field.text().await.unwrap_or_default(),
            "background" => self.background = field.text().await.unwrap_or_default(),
            _ => {}
        }
        Ok(())
    }

    fn finalize(&mut self) -> Result<(), String> {
        if self.name.trim().is_empty() {
            return Err("Name is required".to_string());
        }
        self.name = self.name.trim().to_string();
        self.origin = self.origin.trim().to_string();
        self.grape = self.grape.trim().to_string();
        self.notes = self.notes.trim().to_string();
        self.background = self.background.trim().to_string();
        Ok(())
    }
}

type WishlistFormData = WineBase;

struct WineScores {
    sweetness: i32,
    acidity: i32,
    tannin: i32,
    alcohol: i32,
    body: i32,
    clarity: i32,
    color_intensity: i32,
    aroma_intensity: i32,
    nose_complexity: i32,
}

impl Default for WineScores {
    fn default() -> Self {
        Self {
            sweetness: 3,
            acidity: 3,
            tannin: 3,
            alcohol: 3,
            body: 3,
            clarity: 3,
            color_intensity: 3,
            aroma_intensity: 3,
            nose_complexity: 3,
        }
    }
}

impl WineScores {
    fn slot(&mut self, field_name: &str) -> Option<&mut i32> {
        Some(match field_name {
            "sweetness" => &mut self.sweetness,
            "acidity" => &mut self.acidity,
            "tannin" => &mut self.tannin,
            "alcohol" => &mut self.alcohol,
            "body" => &mut self.body,
            "clarity" => &mut self.clarity,
            "color_intensity" => &mut self.color_intensity,
            "aroma_intensity" => &mut self.aroma_intensity,
            "nose_complexity" => &mut self.nose_complexity,
            _ => return None,
        })
    }

    fn clamp_all(&mut self) {
        for v in [
            &mut self.sweetness,
            &mut self.acidity,
            &mut self.tannin,
            &mut self.alcohol,
            &mut self.body,
            &mut self.clarity,
            &mut self.color_intensity,
            &mut self.aroma_intensity,
            &mut self.nose_complexity,
        ] {
            *v = (*v).clamp(1, 5);
        }
    }
}

struct WineFormData {
    base: WineBase,
    scores: WineScores,
}

impl<'a> From<&'a WineFormData> for crate::db::WineInput<'a> {
    fn from(data: &'a WineFormData) -> Self {
        Self {
            name: &data.base.name,
            origin: &data.base.origin,
            grape: &data.base.grape,
            notes: &data.base.notes,
            sweetness: data.scores.sweetness,
            acidity: data.scores.acidity,
            tannin: data.scores.tannin,
            alcohol: data.scores.alcohol,
            body: data.scores.body,
            clarity: data.scores.clarity,
            color_intensity: data.scores.color_intensity,
            aroma_intensity: data.scores.aroma_intensity,
            nose_complexity: data.scores.nose_complexity,
            background: &data.base.background,
        }
    }
}

async fn parse_wine_multipart(mut multipart: Multipart) -> Result<WineFormData, String> {
    let mut base = WineBase::default();
    let mut scores = WineScores::default();

    while let Ok(Some(field)) = multipart.next_field().await {
        let field_name = field.name().unwrap_or("").to_string();
        if WineBase::owns(&field_name) {
            base.apply_field(&field_name, field).await?;
        } else if let Some(slot) = scores.slot(&field_name) {
            *slot = field.text().await.unwrap_or_default().parse().unwrap_or(3);
        }
    }

    base.finalize()?;
    scores.clamp_all();
    Ok(WineFormData { base, scores })
}

async fn parse_wishlist_multipart(mut multipart: Multipart) -> Result<WishlistFormData, String> {
    let mut base = WineBase::default();
    while let Ok(Some(field)) = multipart.next_field().await {
        let field_name = field.name().unwrap_or("").to_string();
        base.apply_field(&field_name, field).await?;
    }
    base.finalize()?;
    Ok(base)
}

// --- Router ---

pub async fn run(host: String, port: u16) {
    use handlers::{admin, public};

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

    let site_url = std::env::var("SITE_URL")
        .unwrap_or_else(|_| "http://localhost:3000".to_string())
        .trim_end_matches('/')
        .to_string();

    let site_title = std::env::var("SITE_TITLE").unwrap_or_else(|_| "Cellar".to_string());
    let site_description = std::env::var("SITE_DESCRIPTION")
        .unwrap_or_else(|_| "Personal wine tasting log".to_string());

    let state = Arc::new(AppState {
        db,
        app_password,
        cookie_secure,
        anthropic_api_key,
        site_url,
        site_title,
        site_description,
    });

    let app = Router::new()
        // Public routes
        .route("/", get(public::get_index))
        .route("/feed.xml", get(public::rss_feed))
        .route("/wines/{short_id}", get(public::get_wine_detail))
        .route("/wines/{short_id}/image", get(public::get_wine_image))
        // Admin auth routes
        .route("/admin/login", get(admin::get_login).post(admin::post_login))
        .route("/admin/logout", get(admin::get_logout))
        // Admin protected routes
        .route("/admin", get(admin::get_admin))
        .route("/admin/new", get(admin::get_new_wine).post(admin::post_new_wine))
        .route(
            "/admin/edit/{short_id}",
            get(admin::get_edit_wine).post(admin::post_edit_wine),
        )
        .route("/admin/delete/{short_id}", post(admin::post_delete_wine))
        // Wishlist
        .route("/wishlist", get(public::get_wishlist))
        .route(
            "/admin/wishlist/new",
            get(admin::get_new_wishlist_wine).post(admin::post_new_wishlist_wine),
        )
        .route(
            "/admin/wishlist/edit/{short_id}",
            get(admin::get_edit_wishlist_wine).post(admin::post_edit_wishlist_wine),
        )
        .route(
            "/admin/wishlist/delete/{short_id}",
            post(admin::post_delete_wishlist_wine),
        )
        .route(
            "/admin/wishlist/promote/{short_id}",
            post(admin::post_promote_wine),
        )
        // Claude vision
        .route("/admin/analyze-image", post(admin::post_analyze_image))
        // Static assets
        .route("/static/{*path}", get(public::serve_static))
        .merge(andromeda_darkmatter_css::router::<Arc<AppState>>())
        .layer(DefaultBodyLimit::max(10 * 1024 * 1024))
        .with_state(state);

    let addr = format!("{}:{}", host, port);
    tracing::info!("Listening on http://{}", addr);

    let listener = tokio::net::TcpListener::bind(&addr).await.unwrap();
    axum::serve(listener, app).await.unwrap();
}
