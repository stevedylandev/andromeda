# Parcels

![cover](https://files.stevedylan.dev/parcels-demo.png)

A minimal package tracking app

>[!NOTE]
>Parcels currently only supports USPS tracking. If you're interested in adding other providers please feel free to open an issue

## Quickstart

```bash
git clone https://github.com/stevedylandev/parcels.git
cd parcels
cp .env.example .env
# Edit .env with your USPS API credentials and app password
cargo build --release
./target/release/parcels
```

You'll need a [USPS Web Tools API](https://developer.usps.com) account to get your `USPS_CLIENT_ID` and `USPS_CLIENT_SECRET`.

### Environment Variables

| Variable | Description | Default |
|---|---|---|
| `APP_PASSWORD` | Password for login authentication | *required* |
| `DATABASE_URL` | SQLite database path (e.g. `sqlite:///app/data/parcels.db`) |
| `USPS_CLIENT_ID` | USPS OAuth2 client ID | *required* |
| `USPS_CLIENT_SECRET` | USPS OAuth2 client secret | *required* |
| `BIND_ADDR` | Server bind address | `0.0.0.0:3012` |
| `COOKIE_SECURE` | Enable HTTPS-only cookies | `false` |

## Overview

I got tired of logging into USPS, so I built this to track my own personal packages. Over time I might add more providers, but it currently gets the job done. Here's a few highlights: 
- Single ~7MB Rust binary
- Averages around ~10MB of Ram usage
- Password authentication 
- Track USPS packages with custom labels
- Delete packages you no longer want to track

## Structure

```
parcels/
├── src/
│   ├── main.rs        # Axum web server, routes, and app state
│   ├── auth.rs        # Password verification and session management
│   ├── db.rs          # SQLite database layer (packages, events, sessions)
│   └── usps.rs        # USPS API integration with OAuth2 token caching
├── templates/         # Askama HTML templates
│   ├── base.html      # Base layout
│   ├── index.html     # Package list
│   ├── detail.html    # Package detail with tracking events
│   ├── add.html       # Add package form
│   └── login.html     # Login page
├── static/            # Fonts, favicons, and images
├── Dockerfile         # Multi-stage build (Rust 1.87 + Debian slim)
└── docker-compose.yml
```

## Deployment

### Docker (recommended)

```bash
git clone https://github.com/stevedylandev/parcels.git
cd parcels
cp .env.example .env
# Edit .env with your credentials
docker compose up -d
```

This will start Parcels on port `3012` with a persistent volume for the SQLite database.

### Binary

```bash
cargo build --release
```

The resulting binary at `./target/release/parcels` is self-contained (~7MB). Copy it to your server along with the `static/` directory and a configured `.env` file, then run it directly.

## License

[MIT](LICENSE)
