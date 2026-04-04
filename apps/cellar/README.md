# Cellar

![cover](https://files.stevedylan.dev/cellar-demo.png)

A minimal wine collection tracker

## Quickstart

```bash
git clone https://github.com/stevedylandev/cellar.git
cd cellar
cp .env.example .env
# Edit .env with your password and Anthropic API key
cargo build --release
./target/release/cellar
```

### Environment Variables

| Variable | Description | Default |
|---|---|---|
| `CELLAR_PASSWORD` | Password for login authentication | `changeme` |
| `CELLAR_DB_PATH` | SQLite database file path | `cellar.sqlite` |
| `ANTHROPIC_API_KEY` | Anthropic API key for AI features | |
| `HOST` | Server bind address | `127.0.0.1` |
| `PORT` | Server port | `3000` |
| `COOKIE_SECURE` | Enable HTTPS-only cookies | `false` |

## Overview

A simple, self-hosted wine collection app built with Rust. Here's a few highlights:
- Single Rust binary with embedded assets
- Password authentication with session cookies
- Add, edit, and delete wines from your collection
- AI-powered tasting notes via Claude
- Pentagon visualizations for wine profiles
- Dark themed UI with Commit Mono font
- SQLite for persistent storage

## Structure

```
cellar/
├── src/
│   ├── main.rs        # App entrypoint, env vars, starts server
│   ├── server.rs      # Axum router, HTTP handlers, and templates
│   ├── auth.rs        # Password verification and session management
│   ├── claude.rs      # Anthropic API integration for tasting notes
│   └── db.rs          # SQLite database layer (wines, sessions)
├── templates/         # Askama HTML templates
│   ├── base.html      # Base layout with header and nav
│   ├── login.html     # Login page
│   ├── index.html     # Wine collection list
│   ├── wine.html      # Single wine display
│   ├── wine_form.html # Add/edit wine form
│   └── admin.html     # Admin page
├── static/            # Favicons, og:image, styles, and webmanifest
├── Dockerfile         # Multi-stage build (Rust + Debian slim)
└── docker-compose.yml
```

## Deployment

### Docker (recommended)

```bash
git clone https://github.com/stevedylandev/cellar.git
cd cellar
cp .env.example .env
# Edit .env with your password and Anthropic API key
docker compose up -d
```

This will start Cellar on port `3000` with a persistent volume for the SQLite database.

### Binary

```bash
cargo build --release
```

The resulting binary at `./target/release/cellar` is self-contained with all assets embedded. Copy it to your server with a configured `.env` file and run it directly.

## License

[MIT](LICENSE)
