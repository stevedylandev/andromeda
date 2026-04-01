use clap::{Parser, Subcommand};
use std::path::PathBuf;

#[derive(Parser)]
#[command(name = "sipp", about = "Snippet manager — TUI, server, and CLI")]
struct Cli {
    /// Remote server URL (e.g. http://localhost:3000)
    #[arg(short, long, env = "SIPP_REMOTE_URL")]
    remote: Option<String>,

    /// API key for authenticated operations
    #[arg(short = 'k', long, env = "SIPP_API_KEY")]
    api_key: Option<String>,

    /// File path to create a snippet from
    #[arg(value_name = "FILE")]
    file: Option<PathBuf>,

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
        #[arg(long, default_value = "localhost")]
        host: String,
    },
    /// Launch the interactive TUI
    Tui {
        /// Remote server URL (e.g. http://localhost:3000)
        #[arg(short, long, env = "SIPP_REMOTE_URL")]
        remote: Option<String>,

        /// API key for authenticated operations
        #[arg(short = 'k', long, env = "SIPP_API_KEY")]
        api_key: Option<String>,
    },
    /// Save remote URL and API key to config file
    Auth,
}

fn main() -> Result<(), Box<dyn std::error::Error>> {
    let cli = Cli::parse();

    match cli.command {
        Some(Commands::Server { port, host }) => {
            let rt = tokio::runtime::Runtime::new()?;
            rt.block_on(sipp_so::server::run(host, port));
        }
        Some(Commands::Tui { remote, api_key }) => {
            sipp_so::tui::run_interactive(remote, api_key)?;
        }
        Some(Commands::Auth) => {
            sipp_so::tui::run_auth()?;
        }
        None => {
            if let Some(file) = cli.file {
                sipp_so::tui::run_file_upload(cli.remote, cli.api_key, file)?;
            } else {
                sipp_so::tui::run_interactive(cli.remote, cli.api_key)?;
            }
        }
    }

    Ok(())
}
