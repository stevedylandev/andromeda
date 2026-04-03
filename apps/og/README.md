# OG

![cover](https://files.stevedylan.dev/og-demo-1.png)

A simple web tool for inspecting Open Graph tags on any URL.

## Quickstart

```bash
git clone https://github.com/stevedylandev/og.git
cd og
cargo build --release
./target/release/og
```

### Environment Variables

| Variable | Description | Default |
|---|---|---|
| `PORT` | Server port | `3000` |

## Overview

A self-hosted Open Graph tag inspector built with Rust. Enter any URL and instantly see its OG metadata. A few highlights:

- Single Rust binary with embedded assets
- Inspects title, description, image, and other OG tags
- Dark themed UI with Commit Mono font
- No database needed — fully stateless

## Structure

```
og/
├── src/
│   ├── main.rs        # Entry point and server startup
│   ├── server.rs      # Axum routes and request handling
│   └── og.rs          # Open Graph tag fetching and parsing
├── templates/         # Askama HTML templates
│   ├── base.html      # Base layout
│   ├── index.html     # Search form
│   └── results.html   # OG tag results display
├── static/            # Fonts, favicons, and styles
├── Dockerfile
└── docker-compose.yml
```

## Deployment

### Railway

[![Deploy on Railway](https://railway.com/button.svg)](https://railway.com/deploy/OdXBt_?referralCode=JGcIp6)

### Docker (recommended)

```bash
git clone https://github.com/stevedylandev/og.git
cd og
docker compose up -d
```

### Binary

```bash
cargo build --release
```

The resulting binary at `./target/release/og` is self-contained with all assets embedded. Copy it to your server and run it directly.

## License

[MIT](LICENSE)
