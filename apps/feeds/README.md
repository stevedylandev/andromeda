# Feeds

![cover](https://feeds.stevedylan.dev/assets/og.png)

Minimal RSS Feeds

## About

Feeds is a minimal RSS reader that mimics the original experience of RSS. It's just a list of posts. No categories, no marking a post read or unread, and there is no in-app reading. With this approach you have to read the post on the authors personal website and experience it in it's original context. While this may not work well if you have loads of news feeds, I personally love it for [my approach to blogs](https://blogfeeds.net).

This app is also MIT open sourced and designed to be self-hosted; fork the code and change it to your liking!

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

## Project Structure

The architecture is intentionally simple:
- **`src/main.rs`** - Axum server with routing, templates, and static asset serving
- **`src/feeds.rs`** - Feed fetching, OPML parsing, and FreshRSS API integration
- **`src/auth.rs`** - Session-based authentication with constant-time password verification
- **`src/models.rs`** - Data structures for feeds and FreshRSS responses
- **`src/templates/`** - Askama HTML templates
- **`assets/`** - Static assets embedded at compile time via `rust-embed`

## Environment Variables

| Variable | Description | Required |
|---|---|---|
| `FRESHRSS_URL` | URL of your FreshRSS instance | For FreshRSS mode |
| `FRESHRSS_USERNAME` | FreshRSS username | For FreshRSS mode |
| `FRESHRSS_PASSWORD` | FreshRSS password | For FreshRSS mode |
| `ADMIN_PASSWORD` | Password for the admin panel | For admin access |
| `COOKIE_SECURE` | Set to `true` for HTTPS environments | No |

## Deployment

Since Feeds compiles to a single binary, deployment is straightforward on any platform.

### Docker

1. Clone the repo

```bash
git clone https://github.com/stevedylandev/feeds
cd feeds
```

2. Build and run the Docker image

```bash
docker build -t feeds .
docker run -p 3000:3000 --env-file .env feeds
```

Or use `docker-compose`

```bash
docker-compose up -d
```

### Railway

1. Fork the repo from GitHub to your own account

2. Login to [Railway](https://railway.com) and create a new project

3. Select Feeds from your repos

4. Railway will auto-detect the Rust project and build it
