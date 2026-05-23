# bookmarks

Personal link saver organized by category.

## Install

**Homebrew:**

```bash
brew install stevedylandev/tap/bookmarks
```

**Curl install (Linux/macOS):**

```bash
curl -fsSL https://raw.githubusercontent.com/stevedylandev/andromeda/main/install.sh | sh -s -- bookmarks
```

**Prebuilt binary:** Grab the right archive from the [releases page](https://github.com/stevedylandev/andromeda/releases?q=bookmarks%2F) and drop the binary somewhere on your `$PATH`.

**From source:**

```bash
git clone https://github.com/stevedylandev/andromeda
cd andromeda/apps/bookmarks
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
| `BOOKMARKS_PASSWORD` | — | Admin panel password |
| `BOOKMARKS_API_KEY` | — | API key for `POST /api/links` |
| `BOOKMARKS_DB_PATH` | `bookmarks.sqlite` | SQLite path |
| `HOST` | `127.0.0.1` | Bind host |
| `PORT` | `3000` | Server port |
| `COOKIE_SECURE` | `false` | Mark session cookie Secure |

## Routes

Public: `GET /`, `GET /api/categories`, `GET /api/links`. Auth: `GET/POST /login`,
`GET /logout`, `/admin`, `/admin/*`. API-key: `POST /api/links`.
