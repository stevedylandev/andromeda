use askama_web::WebTemplate;
use axum::{
    extract::{Form, Multipart, Path, Query, State},
    http::{HeaderValue, StatusCode},
    response::{Html, IntoResponse, Redirect, Response},
};
use std::io::{Cursor, Read};
use std::sync::Arc;
use zip::ZipArchive;

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
    let slug = derive_slug(title, attrs.slug.trim());

    let status = if form.action == "publish" { "published" } else { "draft" };
    let lang = if attrs.lang.trim().is_empty() { "en" } else { attrs.lang.trim() };
    let published_date = if attrs.published_date.trim().is_empty() {
        now_datetime()
    } else {
        attrs.published_date.trim().to_string()
    };

    let input = db::PostInput {
        title: opt_str(title),
        slug: &slug,
        content: &form.content,
        status,
        alias: opt_str(&attrs.alias),
        canonical_url: None,
        published_date: Some(&published_date),
        meta_description: opt_str(&attrs.meta_description),
        meta_image: opt_str(&attrs.meta_image),
        lang,
        tags: opt_str(&attrs.tags),
    };
    match db::create_post(&state.db, &input) {
        Ok(_) => Redirect::to("/admin").into_response(),
        Err(e) => {
            tracing::error!("Failed to create post: {}", e);
            Redirect::to("/admin/posts/new?error=Failed+to+create+post").into_response()
        }
    }
}

fn derive_slug(title: &str, slug: &str) -> String {
    if !slug.is_empty() {
        return slug.to_string();
    }
    let from_title = slugify(title);
    if !from_title.is_empty() {
        return from_title;
    }
    nanoid::nanoid!(10)
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
    let slug = derive_slug(title, attrs.slug.trim());

    let status = if form.action == "publish" { "published" } else { "draft" };
    let lang = if attrs.lang.trim().is_empty() { "en" } else { attrs.lang.trim() };
    let published_date = if attrs.published_date.trim().is_empty() {
        None
    } else {
        Some(attrs.published_date.trim().to_string())
    };

    let input = db::PostInput {
        title: opt_str(title),
        slug: &slug,
        content: &form.content,
        status,
        alias: opt_str(&attrs.alias),
        canonical_url: None,
        published_date: published_date.as_deref(),
        meta_description: opt_str(&attrs.meta_description),
        meta_image: opt_str(&attrs.meta_image),
        lang,
        tags: opt_str(&attrs.tags),
    };
    match db::update_post(&state.db, &short_id, &input) {
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
    let blog_title = get_setting_or_default(&state.db, "blog_title");
    let blog_description = get_setting_or_default(&state.db, "blog_description");
    let intro_content = get_setting_or_default(&state.db, "intro_content");
    let nav_links = get_setting_or_default(&state.db, "nav_links");
    let custom_css = get_setting_or_default(&state.db, "custom_css");
    let favicon_url = get_setting_or_default(&state.db, "favicon_url");
    let og_image_url = get_setting_or_default(&state.db, "og_image_url");
    let custom_header = get_setting_or_default(&state.db, "custom_header");
    let custom_footer = get_setting_or_default(&state.db, "custom_footer");
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

    let backend = if let Some(r2) = &state.r2 {
        if let Err(e) = r2.put_object(&stored_name, &content_type, bytes.clone()).await {
            tracing::error!("Failed to upload to R2: {}", e);
            return Redirect::to("/admin/files?error=Failed+to+save+file").into_response();
        }
        "r2"
    } else {
        let path = std::path::PathBuf::from(&state.uploads_dir).join(&stored_name);
        if let Err(e) = tokio::fs::write(&path, &bytes).await {
            tracing::error!("Failed to write file: {}", e);
            return Redirect::to("/admin/files?error=Failed+to+save+file").into_response();
        }
        "local"
    };

    match db::create_file(&state.db, &stored_name, &original_name, &content_type, bytes.len() as i64, backend) {
        Ok(_) => Redirect::to("/admin/files?success=true").into_response(),
        Err(e) => {
            tracing::error!("Failed to record file: {}", e);
            if backend == "r2" {
                if let Some(r2) = &state.r2 {
                    if let Err(e) = r2.delete_object(&stored_name).await {
                        tracing::warn!("Failed to roll back R2 upload: {}", e);
                    }
                }
            } else {
                let path = std::path::PathBuf::from(&state.uploads_dir).join(&stored_name);
                let _ = tokio::fs::remove_file(&path).await;
            }
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
            if file.storage_backend == "r2" {
                if let Some(r2) = &state.r2 {
                    if let Err(e) = r2.delete_object(&file.filename).await {
                        tracing::warn!("Failed to delete file from R2: {}", e);
                    }
                } else {
                    tracing::warn!(
                        "File {} stored in R2 but R2 not configured; skipping remote delete",
                        file.filename
                    );
                }
            } else {
                let path = std::path::PathBuf::from(&state.uploads_dir).join(&file.filename);
                if let Err(e) = tokio::fs::remove_file(&path).await {
                    tracing::warn!("Failed to delete file from disk: {}", e);
                }
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
        let markdown_files: Vec<_> = posts
            .iter()
            .map(|p| (format!("{}.md", p.slug), post_to_markdown(p)))
            .collect();
        let entries: Vec<_> = markdown_files
            .iter()
            .map(|(name, content)| (name.clone(), content.as_bytes()))
            .collect();
        build_zip(&entries, zip::CompressionMethod::Deflated)
    })
    .await;

    match result {
        Ok(bytes) => zip_response(bytes, "posts.zip"),
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
        let entries: Vec<_> = file_data
            .iter()
            .map(|(name, bytes)| (name.clone(), bytes.as_slice()))
            .collect();
        build_zip(&entries, zip::CompressionMethod::Stored)
    })
    .await;

    match result {
        Ok(bytes) => zip_response(bytes, "uploads.zip"),
        Err(e) => {
            tracing::error!("Failed to create uploads zip: {}", e);
            (StatusCode::INTERNAL_SERVER_ERROR, "Export failed").into_response()
        }
    }
}

// --- Import handlers ---

const IMPORT_MAX_BYTES: usize = 50 * 1024 * 1024;

pub async fn admin_import_form(
    _session: auth::AuthSession,
    Query(q): Query<FlashQuery>,
) -> Response {
    WebTemplate(AdminImportTemplate {
        error: q.error,
        imported: q.imported,
        skipped: q.skipped,
    })
    .into_response()
}

pub async fn admin_import_posts(
    _session: auth::AuthSession,
    State(state): State<Arc<AppState>>,
    mut multipart: Multipart,
) -> Response {
    let mut zip_data: Option<Vec<u8>> = None;
    while let Ok(Some(field)) = multipart.next_field().await {
        if field.name() == Some("zip") {
            match field.bytes().await {
                Ok(bytes) => zip_data = Some(bytes.to_vec()),
                Err(e) => {
                    tracing::error!("Failed to read import upload: {}", e);
                    return Redirect::to("/admin/import?error=Failed+to+read+upload").into_response();
                }
            }
        }
    }

    let bytes = match zip_data {
        Some(b) => b,
        None => return Redirect::to("/admin/import?error=No+zip+provided").into_response(),
    };
    if bytes.len() > IMPORT_MAX_BYTES {
        return Redirect::to("/admin/import?error=Zip+exceeds+50MB+limit").into_response();
    }

    let db = state.db.clone();
    let result = tokio::task::spawn_blocking(move || process_import_zip(&db, &bytes)).await;

    match result {
        Ok(Ok(summary)) => Redirect::to(&format!(
            "/admin/import?imported={}&skipped={}",
            summary.imported, summary.skipped
        ))
        .into_response(),
        Ok(Err(e)) => {
            tracing::error!("Import failed: {}", e);
            Redirect::to("/admin/import?error=Invalid+zip+archive").into_response()
        }
        Err(e) => {
            tracing::error!("Import join error: {}", e);
            Redirect::to("/admin/import?error=Server+error").into_response()
        }
    }
}

struct ImportSummary {
    imported: u32,
    skipped: u32,
}

fn process_import_zip(db: &db::Db, bytes: &[u8]) -> Result<ImportSummary, String> {
    let mut archive = ZipArchive::new(Cursor::new(bytes))
        .map_err(|e| format!("Bad zip: {}", e))?;

    let mut imported = 0u32;
    let mut skipped = 0u32;

    for i in 0..archive.len() {
        let mut file = match archive.by_index(i) {
            Ok(f) => f,
            Err(e) => {
                tracing::warn!("Skipping zip entry {}: {}", i, e);
                continue;
            }
        };
        if file.is_dir() {
            continue;
        }
        let name = match file.enclosed_name() {
            Some(p) => p.to_string_lossy().into_owned(),
            None => continue,
        };
        if name.starts_with("__MACOSX/") {
            continue;
        }
        let basename = std::path::Path::new(&name)
            .file_name()
            .and_then(|s| s.to_str())
            .unwrap_or("");
        if basename.is_empty() || basename.starts_with('.') {
            continue;
        }
        let lower = basename.to_lowercase();
        if !(lower.ends_with(".md") || lower.ends_with(".markdown")) {
            continue;
        }

        let mut raw = String::new();
        if let Err(e) = file.read_to_string(&mut raw) {
            tracing::warn!("Skipping {}: read error {}", name, e);
            continue;
        }

        if !import_one(db, basename, &raw, &mut imported, &mut skipped) {
            skipped += 1;
        }
    }

    Ok(ImportSummary { imported, skipped })
}

fn import_one(
    db: &db::Db,
    basename: &str,
    raw: &str,
    imported: &mut u32,
    skipped: &mut u32,
) -> bool {
    let (frontmatter, body) = split_frontmatter(raw);
    let attrs = parse_attributes(frontmatter.unwrap_or(""));

    let title = if attrs.title.trim().is_empty() {
        title_from_filename(basename)
    } else {
        attrs.title.trim().to_string()
    };

    let slug = derive_slug(&title, attrs.slug.trim());
    if slug.is_empty() {
        return false;
    }

    match db::get_post_by_slug(db, &slug) {
        Ok(Some(_)) => {
            *skipped += 1;
            return true;
        }
        Ok(None) => {}
        Err(e) => {
            tracing::warn!("DB error checking slug {}: {}", slug, e);
            return false;
        }
    }

    let status = if attrs.status.trim().eq_ignore_ascii_case("published") {
        "published"
    } else {
        "draft"
    };
    let lang = if attrs.lang.trim().is_empty() {
        "en"
    } else {
        attrs.lang.trim()
    };
    let published_date = if attrs.published_date.trim().is_empty() {
        now_datetime()
    } else {
        attrs.published_date.trim().to_string()
    };

    let input = db::PostInput {
        title: opt_str(&title),
        slug: &slug,
        content: body,
        status,
        alias: opt_str(&attrs.alias),
        canonical_url: None,
        published_date: Some(&published_date),
        meta_description: opt_str(&attrs.meta_description),
        meta_image: opt_str(&attrs.meta_image),
        lang,
        tags: opt_str(&attrs.tags),
    };
    match db::create_post(db, &input) {
        Ok(_) => {
            *imported += 1;
            true
        }
        Err(e) => {
            tracing::warn!("Failed to insert {}: {}", slug, e);
            false
        }
    }
}

fn split_frontmatter(content: &str) -> (Option<&str>, &str) {
    let trimmed = content.trim_start_matches('\u{feff}');
    let after_open = if let Some(rest) = trimmed.strip_prefix("---\n") {
        rest
    } else if let Some(rest) = trimmed.strip_prefix("---\r\n") {
        rest
    } else {
        return (None, content);
    };
    for sep in ["\r\n---\r\n", "\r\n---\n", "\n---\r\n", "\n---\n"] {
        if let Some((fm, rest)) = after_open.split_once(sep) {
            let body = rest.trim_start_matches(['\r', '\n']);
            return (Some(fm), body);
        }
    }
    if let Some(fm) = after_open.strip_suffix("\n---").or_else(|| after_open.strip_suffix("\r\n---")) {
        return (Some(fm), "");
    }
    (None, content)
}

fn title_from_filename(name: &str) -> String {
    let stem = name.rsplit_once('.').map(|(s, _)| s).unwrap_or(name);
    let cleaned: String = stem
        .chars()
        .map(|c| if c == '-' || c == '_' { ' ' } else { c })
        .collect();
    let trimmed = cleaned.trim();
    let mut chars = trimmed.chars();
    match chars.next() {
        Some(first) => first.to_uppercase().collect::<String>() + chars.as_str(),
        None => String::new(),
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn split_frontmatter_basic() {
        let input = "---\ntitle: Hello\nslug: hello\n---\n# Body\n";
        let (fm, body) = split_frontmatter(input);
        assert_eq!(fm, Some("title: Hello\nslug: hello"));
        assert_eq!(body, "# Body\n");
    }

    #[test]
    fn split_frontmatter_crlf() {
        let input = "---\r\ntitle: Hi\r\n---\r\nbody\r\n";
        let (fm, body) = split_frontmatter(input);
        assert_eq!(fm, Some("title: Hi"));
        assert_eq!(body, "body\r\n");
    }

    #[test]
    fn split_frontmatter_no_fence() {
        let (fm, body) = split_frontmatter("# Just markdown\n\ncontent");
        assert!(fm.is_none());
        assert_eq!(body, "# Just markdown\n\ncontent");
    }

    #[test]
    fn split_frontmatter_strips_bom() {
        let input = "\u{feff}---\ntitle: Hi\n---\nbody";
        let (fm, body) = split_frontmatter(input);
        assert_eq!(fm, Some("title: Hi"));
        assert_eq!(body, "body");
    }

    #[test]
    fn title_from_filename_replaces_separators() {
        assert_eq!(title_from_filename("my-cool-post.md"), "My cool post");
        assert_eq!(title_from_filename("hello_world.markdown"), "Hello world");
        assert_eq!(title_from_filename("noext"), "Noext");
    }

    #[test]
    fn parse_attributes_picks_up_status() {
        let attrs = parse_attributes("title: T\nstatus: published\n");
        assert_eq!(attrs.title, "T");
        assert_eq!(attrs.status, "published");
    }
}
