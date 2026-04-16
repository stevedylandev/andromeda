mod app;
mod editor;
mod events;
mod render;

use crate::backend::Backend;
use crate::config;
use app::App;
use arboard::Clipboard;
use crossterm::event::{self, Event};
use ratatui::DefaultTerminal;
use std::path::PathBuf;
use std::time::Duration;

fn db_path() -> String {
    std::env::var("JOTTS_DB_PATH").unwrap_or_else(|_| "jotts.sqlite".to_string())
}

fn resolve_backend(
    remote: Option<String>,
    api_key: Option<String>,
) -> Result<(Backend, bool, Option<String>), Box<dyn std::error::Error>> {
    if let Some(url) = remote {
        return Ok((Backend::remote(url.clone(), api_key), true, Some(url)));
    }

    if !std::path::Path::new(&db_path()).exists() {
        let cfg = config::load_config();
        let url = cfg
            .remote_url
            .unwrap_or_else(|| "http://localhost:3000".to_string());
        let api_key = api_key.or(cfg.api_key);
        return Ok((Backend::remote(url.clone(), api_key), true, Some(url)));
    }

    Ok((
        Backend::local(),
        false,
        Some("http://localhost:3000".to_string()),
    ))
}

pub fn run_file_upload(
    remote: Option<String>,
    api_key: Option<String>,
    file: PathBuf,
) -> Result<(), Box<dyn std::error::Error>> {
    let (backend, _, remote_url) = resolve_backend(remote, api_key)?;

    let title = file
        .file_stem()
        .ok_or("Invalid file path")?
        .to_string_lossy()
        .to_string();
    let content = std::fs::read_to_string(&file)
        .map_err(|e| format!("Failed to read file: {}", e))?;
    let note = backend
        .create_note(&title, &content)
        .map_err(|e| format!("{}", e))?;
    let link = match &remote_url {
        Some(url) => format!("{}/notes/{}", url.trim_end_matches('/'), note.short_id),
        None => note.short_id.clone(),
    };
    println!("{}", link);
    if let Ok(mut clipboard) = Clipboard::new() {
        let _ = clipboard.set_text(&link);
        println!("\u{2714} Copied to clipboard!");
    }
    Ok(())
}

pub fn run_auth() -> Result<(), Box<dyn std::error::Error>> {
    use std::io::{self, Write};

    print!("Remote URL: ");
    io::stdout().flush()?;
    let mut remote_url = String::new();
    io::stdin().read_line(&mut remote_url)?;
    let remote_url = remote_url.trim().to_string();

    print!("API Key: ");
    io::stdout().flush()?;
    let api_key = rpassword::read_password()?;
    let api_key = api_key.trim().to_string();

    let cfg = config::Config {
        remote_url: if remote_url.is_empty() {
            None
        } else {
            Some(remote_url)
        },
        api_key: if api_key.is_empty() {
            None
        } else {
            Some(api_key)
        },
    };

    config::save_config(&cfg)?;
    println!("Config saved to {}", config::config_path().display());
    Ok(())
}

pub fn run_interactive(
    remote: Option<String>,
    api_key: Option<String>,
) -> Result<(), Box<dyn std::error::Error>> {
    let (backend, is_remote, remote_url) = resolve_backend(remote, api_key)?;

    let notes = match backend.list_notes() {
        Ok(n) => n,
        Err(e) => {
            eprintln!("Failed to load notes: {}", e);
            Vec::new()
        }
    };

    ratatui::run(|terminal| run_app(terminal, App::new(notes, is_remote, remote_url), &backend))
}

fn run_app(
    terminal: &mut DefaultTerminal,
    mut app: App,
    backend: &Backend,
) -> Result<(), Box<dyn std::error::Error>> {
    while !app.should_quit {
        app.clear_expired_status();

        let content_line_count = app
            .selected_note()
            .map(|n| n.content.lines().count() as u16)
            .unwrap_or(0);

        terminal.draw(|frame| render::draw(frame, &mut app))?;

        if event::poll(Duration::from_millis(100))?
            && let Event::Key(key) = event::read()?
        {
            events::handle_key(terminal, &mut app, backend, key, content_line_count)?;
        }
    }

    Ok(())
}
