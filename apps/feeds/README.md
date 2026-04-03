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
git clone https://github.com/stevedylandev/feeds
cd feeds
cargo build
```

3. Run the dev server

```bash
cargo run
# Server running on http://localhost:3000
```

### Environment Variables

| Variable | Description | Default |
|---|---|---|
| `FRESHRSS_URL` | URL of your FreshRSS instance | — |
| `FRESHRSS_USERNAME` | FreshRSS username | — |
| `FRESHRSS_PASSWORD` | FreshRSS password | — |
| `ADMIN_PASSWORD` | Password for the admin panel | — |
| `COOKIE_SECURE` | Enable HTTPS-only cookies | `false` |

## Overview

Feeds is a minimal RSS reader that mimics the original experience of RSS. It's just a list of posts. No categories, no marking a post read or unread, and there is no in-app reading. With this approach you have to read the post on the author's personal website and experience it in its original context. A few highlights:

- Single Rust binary with embedded assets
- Multiple feed sources: URL params, OPML file, or FreshRSS API
- Password-protected admin panel for managing subscriptions
- Feeds API with JSON and OPML export
- Dark themed UI with Commit Mono font

## Usage

There are several built-in ways to source RSS feeds.

### URL Query Param

Once you have the app running you can add the following to the URL to source an RSS feed:

```
?url=https://bearblog.dev/discover/feed/
```

You can also add multiple URLs by using commas to separate them:

```
?urls=https://bearblog.dev/discover/feed/,https://bearblog.stevedylan.dev/feed/
```

### OPML File

If you save a `feeds.opml` file in the root of the project the app will automatically source it and fetch the posts for the feeds inside.

### FreshRSS API

If neither of the above are provided the app will default to using a FreshRSS API instance. Simply run the following command:

```bash
cp .env.sample .env
```

Then fill in the environment variables:

```
FRESHRSS_URL=
FRESHRSS_USERNAME=
FRESHRSS_PASSWORD=
```

### Admin Panel

Feeds includes a password-protected admin panel at `/admin` for managing your FreshRSS subscriptions. Set the `ADMIN_PASSWORD` environment variable to enable it:

```
ADMIN_PASSWORD=your_secret_password
```

From the admin panel you can view your current subscriptions and add new feeds directly to your FreshRSS instance.

### Feeds API

The `/feeds` endpoint exports your FreshRSS subscriptions in JSON or OPML format:

```
/feeds?format=json
/feeds?format=opml
```

## Structure

```
feeds/
├── src/
│   ├── main.rs        # Axum server with routing, templates, and static asset serving
│   ├── feeds.rs       # Feed fetching, OPML parsing, and FreshRSS API integration
│   ├── auth.rs        # Session-based authentication with constant-time password verification
│   └── models.rs      # Data structures for feeds and FreshRSS responses
├── templates/         # Askama HTML templates
├── assets/            # Static assets embedded at compile time via rust-embed
├── Dockerfile
└── docker-compose.yml
```

## Deployment

Since Feeds compiles to a single binary, deployment is straightforward on any platform.

### Railway

[![Deploy on Railway](https://railway.com/button.svg)](https://railway.com/deploy/Ezvmhx?referralCode=JGcIp6)

### Docker (recommended)

```bash
git clone https://github.com/stevedylandev/feeds
cd feeds
cp .env.sample .env
# Edit .env with your credentials
docker compose up -d
```

### Binary

```bash
cargo build --release
```

The resulting binary at `./target/release/feeds` is self-contained with all assets embedded. Copy it to your server with a configured `.env` file and run it directly.

## License

[MIT](LICENSE)
