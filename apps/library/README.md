# library

Personal book tracker with Google Books search.

## Install

**Homebrew:**

```bash
brew install stevedylandev/tap/library
```

**Prebuilt binary:** Grab the right archive from the [releases page](https://github.com/stevedylandev/andromeda/releases?q=library%2F) and drop the binary somewhere on your `$PATH`.

**From source:**

```bash
git clone https://github.com/stevedylandev/andromeda
cd andromeda/apps/library
go build .
```

## Quickstart

```bash
cp .env.example .env
go run .
```

### Environment Variables

| Variable | Default | Description |
|---|---|---|
| `ADMIN_PASSWORD` | `changeme` | Admin login password |
| `LIBRARY_DB_PATH` | `library.sqlite` | SQLite path |
| `GOOGLE_BOOKS_API_KEY` | — | Optional Google Books API key |
| `BASE_URL` | `http://localhost:3000` | Public base URL (OG tags) |
| `HOST` | `0.0.0.0` | Bind host |
| `PORT` | `3000` | Server port |
| `COOKIE_SECURE` | `false` | Mark session cookie Secure |
| `LIBRARY_DISPLAY_MODE` | `inline` | `inline` or `nav` |
