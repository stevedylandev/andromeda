# Andromeda

![cover](https://files.stevedylan.dev/andromeda-cover.png)

A Rust workspace of minimal, self-hosted web apps. Each app compiles to a single binary powered by Axum, SQLite, and Askama templates.

## Apps

| App | Description |
|---|---|
| [**Sipp**](apps/sipp) | Minimal code sharing with web UI and TUI |
| [**Feeds**](apps/feeds) | Minimal RSS reader with FreshRSS and OPML support |
| [**Parcels**](apps/parcels) | Minimal package tracking (USPS) |
| [**Jotts**](apps/jotts) | Minimal markdown notes app |
| [**OG**](apps/og) | Open Graph tag inspector |
| [**Shrink**](apps/shrink) | Image compression and resizing |

## Shared Crates

| Crate | Description |
|---|---|
| [`andromeda-auth`](crates/auth) | Session-based password authentication |

## Stack

- **Axum** - web framework
- **SQLite** (rusqlite) - storage
- **Askama** - HTML templates
- **rust-embed** - embedded static assets
- **tokio** - async runtime

## Getting Started

```bash
# Build all apps
cargo build --release

# Run a specific app
cargo run -p sipp -- server --port 3000
cargo run -p feeds
cargo run -p parcels
cargo run -p jotts
cargo run -p og
cargo run -p shrink
```

Each app has its own README with detailed setup, environment variables, and deployment instructions.

## License

[MIT](LICENSE)
