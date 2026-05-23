# jotts

Minimal markdown notes. Single binary with subcommands:

- `jotts` — launch TUI (default, no args).
- `jotts tui [--remote URL --api-key KEY]` — TUI editor (Bubble Tea).
- `jotts server` — HTTP server (web UI + JSON API).
- `jotts auth` — save remote URL + API key to config.
- `jotts <file.md>` — upload file as a new note via the JSON API.

## Install

**Homebrew:**

```bash
brew install stevedylandev/tap/jotts
```

**Curl install (Linux/macOS):**

```bash
curl -fsSL https://raw.githubusercontent.com/stevedylandev/andromeda/main/install.sh | sh -s -- jotts
```

**PowerShell (Windows):**

```powershell
& ([scriptblock]::Create((irm https://raw.githubusercontent.com/stevedylandev/andromeda/main/install.ps1))) jotts
```

**Prebuilt binary:** Grab the right archive from the [releases page](https://github.com/stevedylandev/andromeda/releases?q=jotts%2F) and drop the binary somewhere on your `$PATH`.

**From source:**

```bash
git clone https://github.com/stevedylandev/andromeda
cd andromeda/apps/jotts
go build .
```

## Stack

- Go stdlib `net/http` + `html/template`
- `modernc.org/sqlite` (pure-Go SQLite, no CGO)
- `github.com/yuin/goldmark` (markdown w/ strikethrough, tables, tasklists)
- Bubble Tea / Lip Gloss / Glamour for the TUI
- `github.com/pkg/browser` and `github.com/atotto/clipboard` for TUI actions

## Quickstart

Server:

```bash
cp .env.example .env
# edit .env with your password
go run . server
```

TUI:

```bash
go run .
```

## Environment variables

| Variable | Description | Default |
|---|---|---|
| `JOTTS_PASSWORD` | Login password | `changeme` |
| `JOTTS_DB_PATH` | SQLite file path | `jotts.sqlite` |
| `HOST` | Bind address | `127.0.0.1` |
| `PORT` | Server port | `3000` |
| `COOKIE_SECURE` | HTTPS-only cookies | `false` |
| `JOTTS_API_KEY` | API key for `/api/notes` (unset = API disabled) | _(unset)_ |

## API

All endpoints require `x-api-key: $JOTTS_API_KEY` header.

- `GET /api/notes` — list notes
- `POST /api/notes` — create `{title, content}`
- `GET /api/notes/{short_id}`
- `PUT /api/notes/{short_id}` — update `{title, content}`
- `DELETE /api/notes/{short_id}`

## Build

```bash
CGO_ENABLED=0 go build -o jotts .
```

Single self-contained binary with all assets embedded.

## Docker

```bash
docker compose up -d
```
