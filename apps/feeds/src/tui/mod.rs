mod app;
mod events;
mod render;

use crate::backend::Backend;
use crate::config;
use app::App;
use crossterm::event::{self, Event};
use ratatui::DefaultTerminal;
use std::time::Duration;

const DEFAULT_REMOTE: &str = "http://localhost:3000";

fn resolve(remote: Option<String>, api_key: Option<String>) -> (String, Option<String>) {
    let cfg = config::load_config();
    let url = remote
        .or(cfg.remote_url)
        .unwrap_or_else(|| DEFAULT_REMOTE.to_string());
    let key = api_key.or(cfg.api_key);
    (url, key)
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
    let (url, key) = resolve(remote, api_key);
    let has_key = key.is_some();
    let backend = Backend::new(url.clone(), key);

    let items = match backend.list_items(100) {
        Ok(i) => i,
        Err(e) => {
            eprintln!("Failed to load items: {e}");
            Vec::new()
        }
    };

    let app = App::new(items, url, has_key);
    ratatui::run(|terminal| run_app(terminal, app, &backend))
}

pub fn run_preview(
    remote: Option<String>,
    api_key: Option<String>,
    feed_url: String,
) -> Result<(), Box<dyn std::error::Error>> {
    let (url, key) = resolve(remote, api_key);
    let has_key = key.is_some();
    let backend = Backend::new(url.clone(), key);

    let items = match backend.preview(&feed_url) {
        Ok(i) if !i.is_empty() => i,
        Ok(_) => {
            // Empty result — try discover.
            match backend.discover(&feed_url) {
                Ok(feeds) if !feeds.is_empty() => {
                    eprintln!("No items at {feed_url}; trying discovered feed: {}", feeds[0]);
                    backend.preview(&feeds[0]).unwrap_or_default()
                }
                _ => Vec::new(),
            }
        }
        Err(e) => {
            // Fetch failed — site URL may not be a feed; try discovery.
            eprintln!("Preview failed ({e}); trying feed discovery");
            match backend.discover(&feed_url) {
                Ok(feeds) if !feeds.is_empty() => {
                    backend.preview(&feeds[0]).unwrap_or_default()
                }
                Ok(_) => return Err("No feeds found at URL".into()),
                Err(e2) => return Err(format!("preview: {e}; discover: {e2}").into()),
            }
        }
    };

    let app = App::new(items, url, has_key).into_preview(feed_url);
    ratatui::run(|terminal| run_app(terminal, app, &backend))
}

fn run_app(
    terminal: &mut DefaultTerminal,
    mut app: App,
    backend: &Backend,
) -> Result<(), Box<dyn std::error::Error>> {
    while !app.should_quit {
        app.clear_expired_status();
        terminal.draw(|frame| render::draw(frame, &mut app))?;

        if event::poll(Duration::from_millis(100))?
            && let Event::Key(key) = event::read()?
        {
            events::handle_key(&mut app, backend, key);
        }
    }
    Ok(())
}
