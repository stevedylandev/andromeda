use axum::{
    extract::Path,
    http::{HeaderValue, StatusCode, header},
    response::{IntoResponse, Response},
    routing::get,
    Router,
};
use rust_embed::Embed;

#[derive(Embed)]
#[folder = "assets/"]
pub struct Assets;

pub fn router<S>() -> Router<S>
where
    S: Clone + Send + Sync + 'static,
{
    Router::new()
        .route("/assets/darkmatter.css", get(css))
        .route("/assets/fonts/{file}", get(font))
        .route("/darkmatter", get(gallery))
        .route("/darkmatter/", get(gallery))
}

async fn css() -> Response {
    serve("darkmatter.css", "text/css; charset=utf-8")
}

async fn gallery() -> Response {
    serve("index.html", "text/html; charset=utf-8")
}

async fn font(Path(file): Path<String>) -> Response {
    let mime = match file.rsplit('.').next().unwrap_or("") {
        "otf" => "font/otf",
        "ttf" => "font/ttf",
        "woff" => "font/woff",
        "woff2" => "font/woff2",
        _ => "application/octet-stream",
    };
    serve(&format!("fonts/{file}"), mime)
}

fn serve(path: &str, mime: &'static str) -> Response {
    match Assets::get(path) {
        Some(f) => (
            StatusCode::OK,
            [(header::CONTENT_TYPE, HeaderValue::from_static(mime))],
            f.data.to_vec(),
        )
            .into_response(),
        None => StatusCode::NOT_FOUND.into_response(),
    }
}
