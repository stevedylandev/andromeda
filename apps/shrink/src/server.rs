use askama::Template;
use axum::{
    Router,
    extract::Multipart,
    http::{StatusCode, header},
    response::{Html, IntoResponse, Response},
    routing::{get, post},
};
use axum::extract::DefaultBodyLimit;
use tower_http::services::ServeDir;

#[derive(Template)]
#[template(path = "index.html")]
struct IndexTemplate;

pub async fn run(host: String, port: u16) {
    let app = Router::new()
        .route("/", get(get_index))
        .route("/compress", post(post_compress))
        .layer(DefaultBodyLimit::max(20 * 1024 * 1024))
        .nest_service("/static", ServeDir::new("static"));

    let addr = format!("{}:{}", host, port);
    tracing::info!("Listening on {}", addr);
    let listener = tokio::net::TcpListener::bind(&addr).await.unwrap();
    axum::serve(listener, app).await.unwrap();
}

async fn get_index() -> impl IntoResponse {
    let html = IndexTemplate.render().unwrap();
    Html(html)
}

async fn post_compress(mut multipart: Multipart) -> Result<Response, (StatusCode, String)> {
    let mut file_data: Option<Vec<u8>> = None;
    let mut quality: u8 = 80;
    let mut width: u32 = 0;
    let mut original_filename: String = "image".to_string();

    while let Ok(Some(field)) = multipart.next_field().await {
        let name = field.name().unwrap_or("").to_string();
        match name.as_str() {
            "file" => {
                if let Some(fname) = field.file_name() {
                    original_filename = fname.to_string();
                }
                let bytes = field
                    .bytes()
                    .await
                    .map_err(|e| (StatusCode::BAD_REQUEST, format!("Failed to read file: {}", e)))?;
                file_data = Some(bytes.to_vec());
            }
            "quality" => {
                let text = field
                    .text()
                    .await
                    .map_err(|e| (StatusCode::BAD_REQUEST, format!("Failed to read quality: {}", e)))?;
                quality = text.parse::<u8>().unwrap_or(80).clamp(1, 100);
            }
            "width" => {
                let text = field
                    .text()
                    .await
                    .map_err(|e| (StatusCode::BAD_REQUEST, format!("Failed to read width: {}", e)))?;
                width = text.parse::<u32>().unwrap_or(0);
            }
            _ => {}
        }
    }

    let file_data = file_data.ok_or((StatusCode::BAD_REQUEST, "No file provided".to_string()))?;

    let result = tokio::task::spawn_blocking(move || compress_image(&file_data, quality, width))
        .await
        .map_err(|e| (StatusCode::INTERNAL_SERVER_ERROR, format!("Task failed: {}", e)))?
        .map_err(|e| (StatusCode::INTERNAL_SERVER_ERROR, format!("Compression failed: {}", e)))?;

    let download_name = build_download_filename(&original_filename, "jpg");

    Ok((
        StatusCode::OK,
        [
            (header::CONTENT_TYPE, "image/jpeg".to_string()),
            (
                header::CONTENT_DISPOSITION,
                format!("attachment; filename=\"{}\"", download_name),
            ),
        ],
        result,
    )
        .into_response())
}

fn compress_image(data: &[u8], quality: u8, width: u32) -> Result<Vec<u8>, String> {
    let img =
        image::load_from_memory(data).map_err(|e| format!("Failed to decode image: {}", e))?;

    let img = if width > 0 && width != img.width() {
        let aspect = img.height() as f64 / img.width() as f64;
        let height = (width as f64 * aspect).round() as u32;
        img.resize(width, height, image::imageops::FilterType::Lanczos3)
    } else {
        img
    };

    let mut output = Vec::new();
    let encoder = image::codecs::jpeg::JpegEncoder::new_with_quality(&mut output, quality);
    img.write_with_encoder(encoder)
        .map_err(|e| format!("JPEG encoding failed: {}", e))?;

    Ok(output)
}

fn build_download_filename(original: &str, new_ext: &str) -> String {
    let stem = std::path::Path::new(original)
        .file_stem()
        .and_then(|s| s.to_str())
        .unwrap_or("compressed");
    format!("{}_compressed.{}", stem, new_ext)
}
