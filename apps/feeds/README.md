# Feeds

![cover](https://feeds.stevedylan.dev/assets/og.png)

Minimal RSS Feeds

## Quickstart

1. Make sure [Rust](https://www.rust-lang.org/tools/install) is installed

```bash
rustc --version
```

2. Clone and build

```bash
git clone https://github.com/stevedylandev/andromeda
cd andromeda
cargo build -p feeds
```

3. Run the dev server

```bash
cargo run -p feeds
# Server running on http://localhost:3000
```

### Environment Variables

| Variable | Description | Default |
|---|---|---|
| `ADMIN_PASSWORD` | Password for the admin panel | — |
| `API_KEY` | Bearer token for the JSON API at `/api/*` | — |
| `BASE_URL` | Public base URL of the app | `http://localhost:3000` |
| `HOST` | Bind address | `0.0.0.0` |
| `PORT` | Bind port | `3000` |
| `DB_PATH` | SQLite database path | `feeds.sqlite` |
| `DEFAULT_POLL_MINUTES` | Background poll interval in minutes (overridable from the admin panel) | `30` |
| `ITEM_CAP_PER_FEED` | Maximum stored items per subscription; older items pruned | `200` |
| `COOKIE_SECURE` | Enable HTTPS-only cookies | `false` |

## Overview

Feeds is a minimal RSS reader that mimics the original experience of RSS. It's just a list of posts. No categories, no marking a post read or unread, and there is no in-app reading. With this approach you have to read the post on the author's personal website and experience it in its original context. A few highlights:

- Single Rust binary with embedded assets
- Local SQLite storage with a background poller (ETag / `If-Modified-Since` aware)
- Password-protected admin panel for managing subscriptions and categories
- OPML import and JSON/OPML export
- Feed discovery from any site URL
- JSON REST API guarded by a Bearer token
- Ad-hoc preview by passing feed URLs as query params
- Dark themed UI with Commit Mono font

## Usage

### Admin Panel

Set `ADMIN_PASSWORD` and visit `/admin/login`. From the admin panel you can:

- Add feeds by URL (title and site URL are auto-detected on first fetch)
- Discover feeds from any site URL
- Import an OPML file
- Organize subscriptions into categories
- Adjust the poll interval

The background poller starts automatically on launch and re-polls every `DEFAULT_POLL_MINUTES` (or the value saved in the admin panel). Items are deduplicated by GUID and each subscription is capped at `ITEM_CAP_PER_FEED`.

### URL Query Param (preview mode)

You can preview any feed without subscribing by passing it via query string:

```
?url=https://bearblog.dev/discover/feed/
?urls=https://bearblog.dev/discover/feed/,https://bearblog.stevedylan.dev/feed/
```

Preview mode bypasses the database and renders whatever the feed returns live.

### Feeds Export

The `/feeds` endpoint exports your subscriptions:

```
/feeds?format=json
/feeds?format=opml
```

### JSON API

Set `API_KEY` to enable programmatic access. All `/api/*` routes accept `Authorization: Bearer <API_KEY>` or a valid admin session cookie.

| Method | Path | Purpose |
|---|---|---|
| `GET` | `/api/items` | List items. Query: `limit`, `unread`, `category_id`, `subscription_id` |
| `POST` | `/api/items/{id}/read` | Mark item read |
| `POST` | `/api/items/{id}/unread` | Mark item unread |
| `GET` | `/api/subscriptions` | List subscriptions |
| `POST` | `/api/subscriptions` | Add subscription. Body: `{feed_url, title?, category_id?, category_name?}` |
| `PATCH` | `/api/subscriptions/{id}` | Update subscription. Body: `{category_id?, category_name?, clear_category?}` |
| `DELETE` | `/api/subscriptions/{id}` | Remove subscription |
| `GET` | `/api/categories` | List categories |
| `POST` | `/api/categories` | Create category. Body: `{name}` |
| `DELETE` | `/api/categories/{id}` | Remove category |
| `POST` | `/api/import/opml` | Import OPML (multipart `file` field) |
| `GET` | `/api/settings` | Get poll interval and item cap |
| `PUT` | `/api/settings` | Update `poll_interval_minutes` (1-1440) |
| `POST` | `/api/discover` | Discover feeds for a site. Body: `{base_url}` |

## Structure

```
feeds/
├── src/
│   ├── main.rs        # Axum server, admin routes, templates, static serving
│   ├── api.rs         # JSON REST API handlers
│   ├── poller.rs      # Background feed poller
│   ├── feeds.rs       # Feed fetching, OPML parsing, feed discovery
│   ├── auth.rs        # Session + API-key guards
│   └── models.rs      # Data structures
├── templates/         # Askama HTML templates
├── static/            # Static assets embedded at compile time via rust-embed
├── Dockerfile
└── docker-compose.yml
```

Subscription and item storage lives in `crates/db/src/feeds.rs` (shared `andromeda-db` crate).

## Deployment

Since Feeds compiles to a single binary, deployment is straightforward on any platform.

### Railway

[![Deploy on Railway](https://railway.com/button.svg)](https://railway.com/deploy/Ezvmhx?referralCode=JGcIp6)

### Docker (recommended)

```bash
git clone https://github.com/stevedylandev/andromeda
cd andromeda/apps/feeds
cp .env.example .env
# Edit .env with your credentials
docker compose up -d
```

Mount a volume at `DB_PATH` to persist the SQLite database.

### Binary

```bash
cargo build --release -p feeds
```

The resulting binary at `./target/release/feeds` is self-contained with all assets embedded. Copy it to your server with a configured `.env` file and run it directly.

## License

[MIT](LICENSE)
