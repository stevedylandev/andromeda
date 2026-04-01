use axum::{
    Router,
    extract::{Form, Path},
    http::{StatusCode, header},
    response::{Html, IntoResponse, Response},
    routing::{get, post},
};
use askama::Template;
use rust_embed::Embed;
use serde::Deserialize;

use crate::og;

const COMMON_TAGS: &[&str] = &[
    "og:title",
    "og:description",
    "og:image",
    "og:url",
    "og:type",
];

#[derive(Embed)]
#[folder = "static/"]
struct StaticAssets;

#[derive(Template)]
#[template(path = "index.html")]
struct IndexTemplate;

#[derive(Template)]
#[template(path = "results.html")]
struct ResultsTemplate {
    url: String,
    error: Option<String>,
    og_image: Option<String>,
    favicon: Option<String>,
    found_tags: Vec<(String, String)>,
    missing_tags: Vec<String>,
    link_tags: Vec<(String, String, String)>,
}

async fn get_index() -> impl IntoResponse {
    Html(IndexTemplate.render().unwrap())
}

#[derive(Deserialize)]
struct CheckForm {
    url: String,
}

async fn post_check(Form(form): Form<CheckForm>) -> Response {
    let mut url = form.url.trim().to_string();

    if url.is_empty() {
        let tmpl = ResultsTemplate {
            url,
            error: Some("Please enter a URL".to_string()),
            og_image: None,
            favicon: None,
            found_tags: Vec::new(),
            missing_tags: Vec::new(),
            link_tags: Vec::new(),
        };
        return Html(tmpl.render().unwrap()).into_response();
    }

    if !url.starts_with("http://") && !url.starts_with("https://") {
        url = format!("https://{url}");
    }

    match og::fetch_og_data(&url).await {
        Ok(result) => {
            let og_image = result.og_tags.get("og:image").cloned();

            let mut found_tags = Vec::new();
            let mut missing_tags = Vec::new();

            for &tag in COMMON_TAGS {
                if let Some(value) = result.og_tags.get(tag) {
                    found_tags.push((tag.to_string(), value.clone()));
                } else {
                    missing_tags.push(tag.to_string());
                }
            }

            for (key, value) in &result.og_tags {
                if !COMMON_TAGS.contains(&key.as_str()) {
                    found_tags.push((key.clone(), value.clone()));
                }
            }

            let link_tags: Vec<(String, String, String)> = result
                .link_tags
                .into_iter()
                .map(|lt| (lt.rel, lt.href, lt.extra))
                .collect();

            let tmpl = ResultsTemplate {
                url,
                error: None,
                og_image,
                favicon: result.favicon,
                found_tags,
                missing_tags,
                link_tags,
            };
            Html(tmpl.render().unwrap()).into_response()
        }
        Err(err) => {
            let tmpl = ResultsTemplate {
                url,
                error: Some(err),
                og_image: None,
                favicon: None,
                found_tags: Vec::new(),
                missing_tags: Vec::new(),
                link_tags: Vec::new(),
            };
            Html(tmpl.render().unwrap()).into_response()
        }
    }
}

async fn static_handler(Path(path): Path<String>) -> Response {
    match StaticAssets::get(&path) {
        Some(file) => {
            let mime = file.metadata.mimetype();
            (
                StatusCode::OK,
                [(header::CONTENT_TYPE, mime)],
                file.data.to_vec(),
            )
                .into_response()
        }
        None => (StatusCode::NOT_FOUND, "Not found").into_response(),
    }
}

pub async fn run() {
    let app = Router::new()
        .route("/", get(get_index))
        .route("/check", post(post_check))
        .route("/static/{*path}", get(static_handler));

    let port = std::env::var("PORT").unwrap_or_else(|_| "3000".to_string());
    let addr = format!("0.0.0.0:{port}");
    tracing::info!("Listening on http://localhost:{port}");
    let listener = tokio::net::TcpListener::bind(&addr).await.unwrap();
    axum::serve(listener, app).await.unwrap();
}
