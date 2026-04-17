use askama_web::WebTemplate;
use axum::{
    extract::{Path, State},
    http::{HeaderValue, StatusCode, Uri},
    response::{Html, IntoResponse, Redirect, Response},
};
use std::sync::Arc;

use super::super::*;
use crate::db;

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

pub async fn public_index(State(state): State<Arc<AppState>>) -> Response {
    let ctx = SiteContext::from_state(&state);
    let blog_description = get_setting_or_default(&state.db, "blog_description");
    let intro_content = get_setting_or_default(&state.db, "intro_content");

    match db::get_published_posts(&state.db) {
        Ok(posts) => {
            let mut intro_html = render_markdown(&intro_content);

            if intro_content.contains("{{latest_posts}}") {
                let latest: Vec<&Post> = posts.iter().take(5).collect();
                let embed_html = render_latest_posts_embed(&latest);
                intro_html = intro_html.replace("<p>{{latest_posts}}</p>", &embed_html);
                intro_html = intro_html.replace("{{latest_posts}}", &embed_html);
            }

            WebTemplate(IndexTemplate {
                blog_title: ctx.blog_title,
                blog_description,
                intro_html,
                posts,
                nav_links: ctx.nav_links,
                favicon_url: ctx.favicon_url,
                og_image_url: ctx.og_image_url,
                site_url: ctx.site_url,
                header_html: ctx.header_html,
                footer_html: ctx.footer_html,
            })
            .into_response()
        }
        Err(e) => {
            tracing::error!("Failed to list posts: {}", e);
            (StatusCode::INTERNAL_SERVER_ERROR, Html("Server error".to_string())).into_response()
        }
    }
}

pub async fn public_post(
    State(state): State<Arc<AppState>>,
    Path(slug): Path<String>,
) -> Response {
    match db::get_post_by_slug(&state.db, &slug) {
        Ok(Some(post)) if post.status == "published" => {
            let ctx = SiteContext::from_state(&state);
            let rendered_content = render_markdown(&post.content);
            WebTemplate(PostTemplate {
                blog_title: ctx.blog_title,
                nav_links: ctx.nav_links,
                post,
                rendered_content,
                favicon_url: ctx.favicon_url,
                og_image_url: ctx.og_image_url,
                site_url: ctx.site_url,
                header_html: ctx.header_html,
                footer_html: ctx.footer_html,
            })
            .into_response()
        }
        Ok(_) => (StatusCode::NOT_FOUND, Html("Not found".to_string())).into_response(),
        Err(e) => {
            tracing::error!("Failed to get post: {}", e);
            (StatusCode::INTERNAL_SERVER_ERROR, Html("Server error".to_string())).into_response()
        }
    }
}

pub async fn public_page(
    State(state): State<Arc<AppState>>,
    Path(slug): Path<String>,
) -> Response {
    match db::get_page_by_slug(&state.db, &slug) {
        Ok(Some(page)) if page.is_published => {
            let ctx = SiteContext::from_state(&state);
            let rendered_content = render_markdown(&page.content);
            WebTemplate(PageTemplate {
                blog_title: ctx.blog_title,
                nav_links: ctx.nav_links,
                page,
                rendered_content,
                favicon_url: ctx.favicon_url,
                og_image_url: ctx.og_image_url,
                site_url: ctx.site_url,
                header_html: ctx.header_html,
                footer_html: ctx.footer_html,
            })
            .into_response()
        }
        Ok(_) => (StatusCode::NOT_FOUND, Html("Not found".to_string())).into_response(),
        Err(e) => {
            tracing::error!("Failed to get page: {}", e);
            (StatusCode::INTERNAL_SERVER_ERROR, Html("Server error".to_string())).into_response()
        }
    }
}

pub async fn public_posts_list(State(state): State<Arc<AppState>>) -> Response {
    let ctx = SiteContext::from_state(&state);

    match db::get_published_posts(&state.db) {
        Ok(posts) => WebTemplate(PostsListTemplate {
            blog_title: ctx.blog_title,
            nav_links: ctx.nav_links,
            posts,
            favicon_url: ctx.favicon_url,
            og_image_url: ctx.og_image_url,
            site_url: ctx.site_url,
            header_html: ctx.header_html,
            footer_html: ctx.footer_html,
        })
        .into_response(),
        Err(e) => {
            tracing::error!("Failed to list posts: {}", e);
            (StatusCode::INTERNAL_SERVER_ERROR, Html("Server error".to_string())).into_response()
        }
    }
}

pub async fn serve_custom_css(State(state): State<Arc<AppState>>) -> Response {
    let css = get_setting_or_default(&state.db, "custom_css");
    (
        StatusCode::OK,
        [(axum::http::header::CONTENT_TYPE, HeaderValue::from_static("text/css"))],
        css,
    )
        .into_response()
}

pub async fn fallback_handler(
    State(state): State<Arc<AppState>>,
    uri: Uri,
) -> Response {
    let path = uri.path().trim_start_matches('/');
    if let Ok(Some(redirect_to)) = db::find_alias_redirect(&state.db, path) {
        return Redirect::permanent(&redirect_to).into_response();
    }
    (StatusCode::NOT_FOUND, Html("Not found".to_string())).into_response()
}

pub async fn serve_uploaded_file(
    State(state): State<Arc<AppState>>,
    Path(filename): Path<String>,
) -> Response {
    if filename.contains("..") || filename.contains('/') || filename.contains('\\') {
        return StatusCode::NOT_FOUND.into_response();
    }

    let path = std::path::PathBuf::from(&state.uploads_dir).join(&filename);
    match tokio::fs::read(&path).await {
        Ok(bytes) => {
            let mime = mime_from_path(&filename);
            (
                StatusCode::OK,
                [(axum::http::header::CONTENT_TYPE, HeaderValue::from_static(mime))],
                bytes,
            )
                .into_response()
        }
        Err(_) => StatusCode::NOT_FOUND.into_response(),
    }
}

fn xml_escape(s: &str) -> String {
    s.replace('&', "&amp;")
        .replace('<', "&lt;")
        .replace('>', "&gt;")
        .replace('"', "&quot;")
        .replace('\'', "&apos;")
}

pub async fn rss_feed(State(state): State<Arc<AppState>>) -> Response {
    let blog_title = get_blog_title(&state.db);
    let blog_description = get_setting_or_default(&state.db, "blog_description");
    let site_url = &state.site_url;

    let posts = match db::get_published_posts(&state.db) {
        Ok(posts) => posts,
        Err(e) => {
            tracing::error!("Failed to get posts for RSS: {}", e);
            return (StatusCode::INTERNAL_SERVER_ERROR, "Server error").into_response();
        }
    };

    let mut items = String::new();
    for post in &posts {
        let link = format!("{}/posts/{}", site_url, xml_escape(&post.slug));
        let title = xml_escape(&post.title);
        let description = match &post.meta_description {
            Some(d) if !d.is_empty() => xml_escape(d),
            _ => {
                let plain: String = post.content.chars().take(200).collect();
                xml_escape(&plain)
            }
        };
        let pub_date = post.published_date.as_deref().unwrap_or(&post.created_at);
        let guid = format!("{}/posts/{}", site_url, xml_escape(&post.slug));

        items.push_str(&format!(
            "    <item>\n      <title>{title}</title>\n      <link>{link}</link>\n      <guid>{guid}</guid>\n      <description>{description}</description>\n      <pubDate>{pub_date}</pubDate>\n    </item>\n"
        ));
    }

    let last_build = posts
        .first()
        .and_then(|p| p.published_date.as_deref())
        .unwrap_or("");

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
        title = xml_escape(&blog_title),
        site_url = site_url,
        desc = xml_escape(&blog_description),
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
