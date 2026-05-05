use clap::{Parser, Subcommand};

#[derive(Parser)]
#[command(name = "feeds", about = "Minimal RSS feed reader — TUI, server, and CLI")]
struct Cli {
    /// Remote server URL (e.g. http://localhost:3000)
    #[arg(short, long, env = "FEEDS_REMOTE_URL")]
    remote: Option<String>,

    /// API key for authenticated operations
    #[arg(short = 'k', long, env = "FEEDS_API_KEY")]
    api_key: Option<String>,

    /// Feed or site URL to preview
    #[arg(value_name = "URL")]
    url: Option<String>,

    #[command(subcommand)]
    command: Option<Commands>,
}

#[derive(Subcommand)]
enum Commands {
    /// Start the web server (default)
    Serve,
    /// Launch the interactive TUI
    Tui {
        #[arg(short, long, env = "FEEDS_REMOTE_URL")]
        remote: Option<String>,
        #[arg(short = 'k', long, env = "FEEDS_API_KEY")]
        api_key: Option<String>,
    },
    /// Save remote URL and API key to config file
    Auth,
}

fn main() -> Result<(), Box<dyn std::error::Error>> {
    dotenvy::dotenv().ok();
    tracing_subscriber::fmt()
        .with_env_filter(
            tracing_subscriber::EnvFilter::try_from_default_env()
                .unwrap_or_else(|_| tracing_subscriber::EnvFilter::new("info,feeds=info")),
        )
        .init();

    let cli = Cli::parse();

    match cli.command {
        Some(Commands::Serve) => {
            let rt = tokio::runtime::Runtime::new()?;
            rt.block_on(feeds::server::run());
        }
        Some(Commands::Tui { remote, api_key }) => {
            feeds::tui::run_interactive(remote, api_key)?;
        }
        Some(Commands::Auth) => {
            feeds::tui::run_auth()?;
        }
        None => {
            if let Some(url) = cli.url {
                feeds::tui::run_preview(cli.remote, cli.api_key, url)?;
            } else {
                let rt = tokio::runtime::Runtime::new()?;
                rt.block_on(feeds::server::run());
            }
        }
    }

    Ok(())
}
