use clap::{Parser, Subcommand};

#[derive(Parser)]
#[command(name = "jotts", about = "Markdown notes — TUI, server, and CLI")]
struct Cli {
    /// Remote server URL (e.g. http://localhost:3000)
    #[arg(short, long, env = "JOTTS_REMOTE_URL")]
    remote: Option<String>,

    /// API key for authenticated operations
    #[arg(short = 'k', long, env = "JOTTS_API_KEY")]
    api_key: Option<String>,

    #[command(subcommand)]
    command: Option<Commands>,
}

#[derive(Subcommand)]
enum Commands {
    /// Start the web server
    Server {
        /// Port to listen on
        #[arg(short, long, default_value_t = 3000)]
        port: u16,

        /// Host to bind to
        #[arg(long, default_value = "127.0.0.1")]
        host: String,
    },
    /// Launch the interactive TUI
    Tui {
        #[arg(short, long, env = "JOTTS_REMOTE_URL")]
        remote: Option<String>,

        #[arg(short = 'k', long, env = "JOTTS_API_KEY")]
        api_key: Option<String>,
    },
    /// Save remote URL and API key to config file
    Auth,
}

fn main() -> Result<(), Box<dyn std::error::Error>> {
    dotenvy::dotenv().ok();
    tracing_subscriber::fmt::init();

    let cli = Cli::parse();

    match cli.command {
        Some(Commands::Server { port, host }) => {
            let rt = tokio::runtime::Runtime::new()?;
            rt.block_on(jotts::server::run(host, port));
        }
        Some(Commands::Tui { remote, api_key }) => {
            jotts::tui::run_interactive(remote, api_key)?;
        }
        Some(Commands::Auth) => {
            jotts::tui::run_auth()?;
        }
        None => {
            jotts::tui::run_interactive(cli.remote, cli.api_key)?;
        }
    }

    Ok(())
}
