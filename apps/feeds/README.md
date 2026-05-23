# feeds

Minimal RSS reader built on the Go standard library plus a SQLite driver and
a feed parser.

## Install

**Homebrew:**

```bash
brew install stevedylandev/tap/feeds
```

**Curl install (Linux/macOS):**

```bash
curl -fsSL https://raw.githubusercontent.com/stevedylandev/andromeda/main/install.sh | sh -s -- feeds
```

**PowerShell (Windows):**

```powershell
& ([scriptblock]::Create((irm https://raw.githubusercontent.com/stevedylandev/andromeda/main/install.ps1))) feeds
```

**Prebuilt binary:** Grab the right archive from the [releases page](https://github.com/stevedylandev/andromeda/releases?q=feeds%2F) and drop the binary somewhere on your `$PATH`.

**From source:**

```bash
git clone https://github.com/stevedylandev/andromeda
cd andromeda/apps/feeds
go build .
```

## Stack

- `net/http`
- `html/template`
- `database/sql`
- `embed`
- `modernc.org/sqlite`
- `github.com/mmcdole/gofeed`

## Run

```bash
cd apps/feeds
go run .
```

Copy `.env.example` to `.env` if you want local config.

## What it includes

- public feed list
- preview mode via `?url=` / `?urls=` (single `?url=*.opml` fetches and
  previews the OPML feed list, up to 5 items per feed)
- admin login with cookie sessions
- add/remove subscriptions and categories
- OPML import (admin form + `POST /api/import/opml`) and export
  (`/feeds?format=opml`)
- JSON API
- background polling with ETag / Last-Modified
- embedded templates and static assets
