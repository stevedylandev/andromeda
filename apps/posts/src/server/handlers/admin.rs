use askama_web::WebTemplate;
use axum::{
    extract::{Form, Multipart, Path, Query, State},
    http::{HeaderValue, StatusCode},
    response::{Html, IntoResponse, Redirect, Response},
};
use std::sync::Arc;

use super::super::*;
use crate::{auth, db};

// --- Auth handlers ---

pub async fn get_login(Query(q): Query<FlashQuery>) -> Response {
    WebTemplate(LoginTemplate { error: q.error }).into_response()
}

pub async fn post_login(
    State(state): State<Arc<AppState>>,
    Form(form): Form<LoginForm>,
) -> Response {
    if !auth::verify_password(&form.password, &state.app_password) {
        return Redirect::to("/admin/login?error=Invalid+password").into_response();
    }

    let token = auth::generate_session_token();

    let expires_at = andromeda_auth::datetime::expiry_datetime_string(7 * 24 * 3600);

    if let Err(e) = db::insert_session(&state.db, &token, &expires_at) {
        tracing::error!("Failed to create session: {}", e);
        return Redirect::to("/admin/login?error=Server+error").into_response();
    }

    let cookie = auth::build_session_cookie(&token, state.cookie_secure);
    let mut resp = Redirect::to("/admin").into_response();
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

// --- Admin post handlers ---

pub async fn admin_index(
    _session: auth::AuthSession,
    State(state): State<Arc<AppState>>,
) -> Response {
    match db::get_all_posts(&state.db) {
        Ok(posts) => WebTemplate(AdminIndexTemplate { posts }).into_response(),
        Err(e) => {
            tracing::error!("Failed to list posts: {}", e);
            (StatusCode::INTERNAL_SERVER_ERROR, Html("Server error".to_string())).into_response()
        }
    }
}

pub async fn admin_new_post(
    _session: auth::AuthSession,
    Query(q): Query<FlashQuery>,
) -> Response {
    WebTemplate(AdminPostFormTemplate {
        post: None,
        error: q.error,
    })
    .into_response()
}

pub async fn admin_create_post(
    _session: auth::AuthSession,
    State(state): State<Arc<AppState>>,
    Form(form): Form<PostForm>,
) -> Response {
    let attrs = parse_attributes(&form.attributes);
    let title = attrs.title.trim();
    if title.is_empty() {
        return Redirect::to("/admin/posts/new?error=Title+is+required").into_response();
    }
    let slug = if attrs.slug.trim().is_empty() {
        slugify(title)
    } else {
        attrs.slug.trim().to_string()
    };

    let status = if form.action == "publish" { "published" } else { "draft" };
    let lang = if attrs.lang.trim().is_empty() { "en" } else { attrs.lang.trim() };
    let published_date = if attrs.published_date.trim().is_empty() {
        now_datetime()
    } else {
        attrs.published_date.trim().to_string()
    };

    match db::create_post(
        &state.db,
        title,
        &slug,
        &form.content,
        status,
        opt_str(&attrs.alias),
        None,
        Some(&published_date),
        opt_str(&attrs.meta_description),
        opt_str(&attrs.meta_image),
        lang,
        opt_str(&attrs.tags),
    ) {
        Ok(_) => Redirect::to("/admin").into_response(),
        Err(e) => {
            tracing::error!("Failed to create post: {}", e);
            Redirect::to("/admin/posts/new?error=Failed+to+create+post").into_response()
        }
    }
}

pub async fn admin_edit_post(
    _session: auth::AuthSession,
    State(state): State<Arc<AppState>>,
    Path(short_id): Path<String>,
    Query(q): Query<FlashQuery>,
) -> Response {
    match db::get_post_by_short_id(&state.db, &short_id) {
        Ok(Some(post)) => WebTemplate(AdminPostFormTemplate {
            post: Some(post),
            error: q.error,
        })
        .into_response(),
        Ok(None) => (StatusCode::NOT_FOUND, Html("Post not found".to_string())).into_response(),
        Err(e) => {
            tracing::error!("Failed to get post: {}", e);
            (StatusCode::INTERNAL_SERVER_ERROR, Html("Server error".to_string())).into_response()
        }
    }
}

pub async fn admin_update_post(
    _session: auth::AuthSession,
    State(state): State<Arc<AppState>>,
    Path(short_id): Path<String>,
    Form(form): Form<PostForm>,
) -> Response {
    let attrs = parse_attributes(&form.attributes);
    let title = attrs.title.trim();
    if title.is_empty() {
        return Redirect::to(&format!("/admin/posts/{}/edit?error=Title+is+required", short_id))
            .into_response();
    }
    let slug = if attrs.slug.trim().is_empty() {
        slugify(title)
    } else {
        attrs.slug.trim().to_string()
    };

    let status = if form.action == "publish" { "published" } else { "draft" };
    let lang = if attrs.lang.trim().is_empty() { "en" } else { attrs.lang.trim() };
    let published_date = if attrs.published_date.trim().is_empty() {
        None
    } else {
        Some(attrs.published_date.trim().to_string())
    };

    match db::update_post(
        &state.db,
        &short_id,
        title,
        &slug,
        &form.content,
        status,
        opt_str(&attrs.alias),
        None,
        published_date.as_deref(),
        opt_str(&attrs.meta_description),
        opt_str(&attrs.meta_image),
        lang,
        opt_str(&attrs.tags),
    ) {
        Ok(Some(_)) => Redirect::to("/admin").into_response(),
        Ok(None) => (StatusCode::NOT_FOUND, Html("Post not found".to_string())).into_response(),
        Err(e) => {
            tracing::error!("Failed to update post: {}", e);
            Redirect::to(&format!("/admin/posts/{}/edit?error=Failed+to+update", short_id))
                .into_response()
        }
    }
}

pub async fn admin_delete_post(
    _session: auth::AuthSession,
    State(state): State<Arc<AppState>>,
    Path(short_id): Path<String>,
) -> Response {
    match db::delete_post(&state.db, &short_id) {
        Ok(_) => Redirect::to("/admin").into_response(),
        Err(e) => {
            tracing::error!("Failed to delete post: {}", e);
            Redirect::to("/admin").into_response()
        }
    }
}

pub async fn admin_toggle_publish(
    _session: auth::AuthSession,
    State(state): State<Arc<AppState>>,
    Path(short_id): Path<String>,
) -> Response {
    match db::toggle_post_status(&state.db, &short_id) {
        Ok(_) => Redirect::to("/admin").into_response(),
        Err(e) => {
            tracing::error!("Failed to toggle post status: {}", e);
            Redirect::to("/admin").into_response()
        }
    }
}

// --- Admin page handlers ---

pub async fn admin_pages(
    _session: auth::AuthSession,
    State(state): State<Arc<AppState>>,
) -> Response {
    match db::get_all_pages(&state.db) {
        Ok(pages) => WebTemplate(AdminPagesTemplate { pages }).into_response(),
        Err(e) => {
            tracing::error!("Failed to list pages: {}", e);
            (StatusCode::INTERNAL_SERVER_ERROR, Html("Server error".to_string())).into_response()
        }
    }
}

pub async fn admin_new_page(
    _session: auth::AuthSession,
    Query(q): Query<FlashQuery>,
) -> Response {
    WebTemplate(AdminPageFormTemplate {
        page: None,
        error: q.error,
    })
    .into_response()
}

const RESERVED_PAGE_SLUGS: &[&str] = &[
    "posts", "admin", "feed.xml", "custom-styles.css", "static", "files",
];

fn is_reserved_page_slug(slug: &str) -> bool {
    RESERVED_PAGE_SLUGS.contains(&slug)
}

pub async fn admin_create_page(
    _session: auth::AuthSession,
    State(state): State<Arc<AppState>>,
    Form(form): Form<PageForm>,
) -> Response {
    let attrs = parse_page_attributes(&form.attributes);
    let title = attrs.title.trim().to_string();
    let slug = attrs.slug.trim().to_string();
    if title.is_empty() || slug.is_empty() {
        return Redirect::to("/admin/pages/new?error=Title+and+slug+are+required").into_response();
    }
    if is_reserved_page_slug(&slug) {
        return Redirect::to("/admin/pages/new?error=That+slug+is+reserved").into_response();
    }

    match db::create_page(&state.db, &title, &slug, &form.content, attrs.is_published, 0) {
        Ok(_) => Redirect::to("/admin/pages").into_response(),
        Err(e) => {
            tracing::error!("Failed to create page: {}", e);
            Redirect::to("/admin/pages/new?error=Failed+to+create+page").into_response()
        }
    }
}

pub async fn admin_edit_page(
    _session: auth::AuthSession,
    State(state): State<Arc<AppState>>,
    Path(short_id): Path<String>,
    Query(q): Query<FlashQuery>,
) -> Response {
    match db::get_page_by_short_id(&state.db, &short_id) {
        Ok(Some(page)) => WebTemplate(AdminPageFormTemplate {
            page: Some(page),
            error: q.error,
        })
        .into_response(),
        Ok(None) => (StatusCode::NOT_FOUND, Html("Page not found".to_string())).into_response(),
        Err(e) => {
            tracing::error!("Failed to get page: {}", e);
            (StatusCode::INTERNAL_SERVER_ERROR, Html("Server error".to_string())).into_response()
        }
    }
}

pub async fn admin_update_page(
    _session: auth::AuthSession,
    State(state): State<Arc<AppState>>,
    Path(short_id): Path<String>,
    Form(form): Form<PageForm>,
) -> Response {
    let attrs = parse_page_attributes(&form.attributes);
    let title = attrs.title.trim().to_string();
    let slug = attrs.slug.trim().to_string();
    if title.is_empty() || slug.is_empty() {
        return Redirect::to(&format!(
            "/admin/pages/{}/edit?error=Title+and+slug+are+required",
            short_id
        ))
        .into_response();
    }
    if is_reserved_page_slug(&slug) {
        return Redirect::to(&format!(
            "/admin/pages/{}/edit?error=That+slug+is+reserved",
            short_id
        ))
        .into_response();
    }

    match db::update_page(&state.db, &short_id, &title, &slug, &form.content, attrs.is_published, 0) {
        Ok(Some(_)) => Redirect::to("/admin/pages").into_response(),
        Ok(None) => (StatusCode::NOT_FOUND, Html("Page not found".to_string())).into_response(),
        Err(e) => {
            tracing::error!("Failed to update page: {}", e);
            Redirect::to(&format!("/admin/pages/{}/edit?error=Failed+to+update", short_id))
                .into_response()
        }
    }
}

pub async fn admin_delete_page(
    _session: auth::AuthSession,
    State(state): State<Arc<AppState>>,
    Path(short_id): Path<String>,
) -> Response {
    match db::delete_page(&state.db, &short_id) {
        Ok(_) => Redirect::to("/admin/pages").into_response(),
        Err(e) => {
            tracing::error!("Failed to delete page: {}", e);
            Redirect::to("/admin/pages").into_response()
        }
    }
}

// --- Admin settings handlers ---

pub async fn admin_get_settings(
    _session: auth::AuthSession,
    State(state): State<Arc<AppState>>,
    Query(q): Query<FlashQuery>,
) -> Response {
    let blog_title = db::get_setting(&state.db, "blog_title").ok().flatten().unwrap_or_default();
    let blog_description = db::get_setting(&state.db, "blog_description").ok().flatten().unwrap_or_default();
    let intro_content = db::get_setting(&state.db, "intro_content").ok().flatten().unwrap_or_default();
    let nav_links = db::get_setting(&state.db, "nav_links").ok().flatten().unwrap_or_default();
    let custom_css = db::get_setting(&state.db, "custom_css").ok().flatten().unwrap_or_default();
    let favicon_url = db::get_setting(&state.db, "favicon_url").ok().flatten().unwrap_or_default();
    let og_image_url = db::get_setting(&state.db, "og_image_url").ok().flatten().unwrap_or_default();
    let custom_header = db::get_setting(&state.db, "custom_header").ok().flatten().unwrap_or_default();
    let custom_footer = db::get_setting(&state.db, "custom_footer").ok().flatten().unwrap_or_default();
    let default_css = Static::get("styles.css")
        .map(|f| String::from_utf8_lossy(&f.data).into_owned())
        .unwrap_or_default();

    WebTemplate(AdminSettingsTemplate {
        blog_title,
        blog_description,
        intro_content,
        nav_links,
        custom_css,
        default_css,
        favicon_url,
        og_image_url,
        custom_header,
        custom_footer,
        success: q.success,
    })
    .into_response()
}

pub async fn admin_post_settings(
    _session: auth::AuthSession,
    State(state): State<Arc<AppState>>,
    Form(form): Form<SettingsForm>,
) -> Response {
    let _ = db::set_setting(&state.db, "blog_title", form.blog_title.trim());
    let _ = db::set_setting(&state.db, "blog_description", form.blog_description.trim());
    let _ = db::set_setting(&state.db, "intro_content", &form.intro_content);
    let _ = db::set_setting(&state.db, "nav_links", &form.nav_links);
    let _ = db::set_setting(&state.db, "custom_css", &form.custom_css);
    let _ = db::set_setting(&state.db, "favicon_url", form.favicon_url.trim());
    let _ = db::set_setting(&state.db, "og_image_url", form.og_image_url.trim());
    let _ = db::set_setting(&state.db, "custom_header", &form.custom_header);
    let _ = db::set_setting(&state.db, "custom_footer", &form.custom_footer);
    Redirect::to("/admin/settings?success=true").into_response()
}

// --- Admin file handlers ---

pub async fn admin_files(
    _session: auth::AuthSession,
    State(state): State<Arc<AppState>>,
    Query(q): Query<FlashQuery>,
) -> Response {
    match db::get_all_files(&state.db) {
        Ok(files) => WebTemplate(AdminFilesTemplate {
            files,
            site_url: state.site_url.clone(),
            error: q.error,
            success: q.success,
        })
        .into_response(),
        Err(e) => {
            tracing::error!("Failed to list files: {}", e);
            (StatusCode::INTERNAL_SERVER_ERROR, Html("Server error".to_string())).into_response()
        }
    }
}

pub async fn admin_upload_file(
    _session: auth::AuthSession,
    State(state): State<Arc<AppState>>,
    mut multipart: Multipart,
) -> Response {
    let mut file_data: Option<(String, String, Vec<u8>)> = None;

    while let Ok(Some(field)) = multipart.next_field().await {
        if field.name() == Some("file") {
            let original_name = field
                .file_name()
                .unwrap_or("upload")
                .to_string();
            let content_type = field
                .content_type()
                .unwrap_or("application/octet-stream")
                .to_string();
            match field.bytes().await {
                Ok(bytes) => {
                    file_data = Some((original_name, content_type, bytes.to_vec()));
                }
                Err(e) => {
                    tracing::error!("Failed to read upload: {}", e);
                    return Redirect::to("/admin/files?error=Failed+to+read+upload").into_response();
                }
            }
        }
    }

    let (original_name, content_type, bytes) = match file_data {
        Some(d) => d,
        None => return Redirect::to("/admin/files?error=No+file+provided").into_response(),
    };

    let max_size: usize = 10 * 1024 * 1024;
    if bytes.len() > max_size {
        return Redirect::to("/admin/files?error=File+exceeds+10MB+limit").into_response();
    }

    let ext = original_name
        .rsplit('.')
        .next()
        .filter(|e| !e.is_empty() && *e != original_name)
        .unwrap_or("");
    let id = nanoid::nanoid!(10);
    let stored_name = if ext.is_empty() {
        id
    } else {
        format!("{}.{}", id, ext)
    };

    let path = std::path::PathBuf::from(&state.uploads_dir).join(&stored_name);
    if let Err(e) = tokio::fs::write(&path, &bytes).await {
        tracing::error!("Failed to write file: {}", e);
        return Redirect::to("/admin/files?error=Failed+to+save+file").into_response();
    }

    match db::create_file(&state.db, &stored_name, &original_name, &content_type, bytes.len() as i64) {
        Ok(_) => Redirect::to("/admin/files?success=true").into_response(),
        Err(e) => {
            tracing::error!("Failed to record file: {}", e);
            let _ = tokio::fs::remove_file(&path).await;
            Redirect::to("/admin/files?error=Failed+to+record+file").into_response()
        }
    }
}

pub async fn admin_delete_file(
    _session: auth::AuthSession,
    State(state): State<Arc<AppState>>,
    Path(short_id): Path<String>,
) -> Response {
    match db::delete_file(&state.db, &short_id) {
        Ok(Some(file)) => {
            let path = std::path::PathBuf::from(&state.uploads_dir).join(&file.filename);
            if let Err(e) = tokio::fs::remove_file(&path).await {
                tracing::warn!("Failed to delete file from disk: {}", e);
            }
            Redirect::to("/admin/files").into_response()
        }
        Ok(None) => Redirect::to("/admin/files").into_response(),
        Err(e) => {
            tracing::error!("Failed to delete file: {}", e);
            Redirect::to("/admin/files").into_response()
        }
    }
}

// --- Download/export handlers ---

pub async fn admin_download_posts(
    _session: auth::AuthSession,
    State(state): State<Arc<AppState>>,
) -> Response {
    let posts = match db::get_all_posts(&state.db) {
        Ok(posts) => posts,
        Err(e) => {
            tracing::error!("Failed to get posts for export: {}", e);
            return (StatusCode::INTERNAL_SERVER_ERROR, "Server error").into_response();
        }
    };

    let result = tokio::task::spawn_blocking(move || {
        let mut buf = std::io::Cursor::new(Vec::new());
        {
            let mut zip = zip::ZipWriter::new(&mut buf);
            let options = zip::write::SimpleFileOptions::default()
                .compression_method(zip::CompressionMethod::Deflated);
            for post in &posts {
                let filename = format!("{}.md", post.slug);
                let mut frontmatter = format!(
                    "---\ntitle: {}\nslug: {}\nstatus: {}",
                    post.title, post.slug, post.status
                );
                if let Some(ref pd) = post.published_date {
                    frontmatter.push_str(&format!("\npublished_date: {}", pd));
                }
                if let Some(ref tags) = post.tags {
                    frontmatter.push_str(&format!("\ntags: {}", tags));
                }
                frontmatter.push_str(&format!("\nlang: {}", post.lang));
                if let Some(ref alias) = post.alias {
                    frontmatter.push_str(&format!("\nalias: {}", alias));
                }
                if let Some(ref meta_image) = post.meta_image {
                    frontmatter.push_str(&format!("\nmeta_image: {}", meta_image));
                }
                if let Some(ref meta_desc) = post.meta_description {
                    frontmatter.push_str(&format!("\ndescription: {}", meta_desc));
                }
                frontmatter.push_str("\n---\n\n");
                let content = format!("{}{}", frontmatter, post.content);
                if let Err(e) = zip.start_file(&filename, options) {
                    tracing::warn!("Failed to add {} to zip: {}", filename, e);
                    continue;
                }
                if let Err(e) = std::io::Write::write_all(&mut zip, content.as_bytes()) {
                    tracing::warn!("Failed to write {} to zip: {}", filename, e);
                }
            }
            let _ = zip.finish();
        }
        buf.into_inner()
    })
    .await;

    match result {
        Ok(bytes) => (
            StatusCode::OK,
            [
                (axum::http::header::CONTENT_TYPE, "application/zip"),
                (
                    axum::http::header::CONTENT_DISPOSITION,
                    "attachment; filename=\"posts.zip\"",
                ),
            ],
            bytes,
        )
            .into_response(),
        Err(e) => {
            tracing::error!("Failed to create posts zip: {}", e);
            (StatusCode::INTERNAL_SERVER_ERROR, "Export failed").into_response()
        }
    }
}

pub async fn admin_download_uploads(
    _session: auth::AuthSession,
    State(state): State<Arc<AppState>>,
) -> Response {
    let files = match db::get_all_files(&state.db) {
        Ok(files) => files,
        Err(e) => {
            tracing::error!("Failed to get files for export: {}", e);
            return (StatusCode::INTERNAL_SERVER_ERROR, "Server error").into_response();
        }
    };

    let uploads_dir = state.uploads_dir.clone();
    let mut file_data: Vec<(String, Vec<u8>)> = Vec::new();
    let mut seen_names = std::collections::HashSet::new();
    for file in &files {
        let path = std::path::PathBuf::from(&uploads_dir).join(&file.filename);
        match tokio::fs::read(&path).await {
            Ok(bytes) => {
                let name = if seen_names.contains(&file.original_name) {
                    format!("{}_{}", file.short_id, file.original_name)
                } else {
                    file.original_name.clone()
                };
                seen_names.insert(file.original_name.clone());
                file_data.push((name, bytes));
            }
            Err(e) => {
                tracing::warn!("Skipping file {} ({}): {}", file.original_name, file.filename, e);
            }
        }
    }

    let result = tokio::task::spawn_blocking(move || {
        let mut buf = std::io::Cursor::new(Vec::new());
        {
            let mut zip = zip::ZipWriter::new(&mut buf);
            let options = zip::write::SimpleFileOptions::default()
                .compression_method(zip::CompressionMethod::Stored);
            for (name, bytes) in &file_data {
                if let Err(e) = zip.start_file(name, options) {
                    tracing::warn!("Failed to add {} to zip: {}", name, e);
                    continue;
                }
                if let Err(e) = std::io::Write::write_all(&mut zip, bytes) {
                    tracing::warn!("Failed to write {} to zip: {}", name, e);
                }
            }
            let _ = zip.finish();
        }
        buf.into_inner()
    })
    .await;

    match result {
        Ok(bytes) => (
            StatusCode::OK,
            [
                (axum::http::header::CONTENT_TYPE, "application/zip"),
                (
                    axum::http::header::CONTENT_DISPOSITION,
                    "attachment; filename=\"uploads.zip\"",
                ),
            ],
            bytes,
        )
            .into_response(),
        Err(e) => {
            tracing::error!("Failed to create uploads zip: {}", e);
            (StatusCode::INTERNAL_SERVER_ERROR, "Export failed").into_response()
        }
    }
}
