# Posts

![cover](https://assets.andromeda.build/posts-demo.png)

A minimal CMS blog with admin interface

## Quickstart

```bash
cd apps/posts
cp .env.example .env
# Edit .env with your password
cargo build --release
./target/release/posts
```

### Environment Variables

| Variable | Description | Default |
|---|---|---|
| `POSTS_PASSWORD` | Password for admin login | `changeme` |
| `POSTS_DB_PATH` | SQLite database file path | `posts.sqlite` |
| `UPLOADS_DIR` | Directory for uploaded files | `uploads` |
| `SITE_URL` | Public URL for RSS feed and links | `http://localhost:3000` |
| `HOST` | Server bind address | `127.0.0.1` |
| `PORT` | Server port | `3000` |
| `COOKIE_SECURE` | Enable HTTPS-only cookies | `false` |

## Overview

A self-hosted blog CMS built with Rust. Here's a few highlights:
- Single Rust binary with embedded assets
- Password authentication with session cookies
- Create, edit, publish, and delete blog posts with markdown
- Static pages with custom navigation links
- File uploads with admin management
- Custom CSS support from the admin panel
- RSS feed at `/feed.xml`
- Dark themed UI with Commit Mono font
- SQLite for persistent storage

## Structure

```
posts/
├── src/
│   ├── main.rs        # App entrypoint, env vars, starts server
│   ├── server.rs      # Axum router, HTTP handlers, and templates
│   ├── auth.rs        # Password verification and session management
│   └── db.rs          # SQLite database layer (posts, pages, files, settings, sessions)
├── templates/         # Askama HTML templates
│   ├── base.html            # Public base layout
│   ├── index.html           # Blog home page
│   ├── post.html            # Single post view
│   ├── posts.html           # Post listing
│   ├── page.html            # Static page view
│   ├── login.html           # Login page
│   ├── admin_base.html      # Admin layout
│   ├── admin_index.html     # Admin dashboard
│   ├── admin_post_form.html # Create/edit post form
│   ├── admin_pages.html     # Admin page listing
│   ├── admin_page_form.html # Create/edit page form
│   ├── admin_files.html     # File upload management
│   └── admin_settings.html  # Blog settings
├── static/            # Favicons, fonts, and styles
├── uploads/           # Uploaded files directory
├── Dockerfile
└── docker-compose.yml
```

## Deployment

### Railway

[![Deploy on Railway](https://railway.com/button.svg)](https://railway.com/deploy/tYtJYp?referralCode=JGcIp6)

### Docker (recommended)

```bash
cd apps/posts
cp .env.example .env
# Edit .env with your password
docker compose up -d
```

This will start Posts on port `3000` with a persistent volume for the SQLite database and uploads.

### Binary

```bash
cargo build --release
```

The resulting binary at `./target/release/posts` is self-contained with all assets embedded. Copy it to your server with a configured `.env` file and run it directly.

## License

[MIT](LICENSE)
