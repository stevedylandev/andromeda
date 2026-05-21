---
name: Andromeda Stack
description: Scaffold a Go CRUD web app using net/http + html/template + modernc.org/sqlite + the shared andromeda crates-go packages (auth, web, config, sqlite, darkmatter). Use when the user wants to build a new Go web server with CRUD operations in the andromeda monorepo.
---

# Go Andromeda Web App

## Overview

Scaffold and build Go CRUD web apps in the andromeda workspace using the
standard library `net/http` mux, `html/template`, `modernc.org/sqlite` (pure
Go, no cgo) and the shared `crates-go/*` packages. Each app ships a single Go
binary with HTML pages, a JSON API, optional session or API key auth, embedded
templates + static assets via `embed.FS`, and Docker deployment.

Apps live under `apps/<name>/`. Each app is its own Go module. Shared packages
live under `crates-go/` and are each their own module, wired in via local
`replace` directives.

Shared crates:

- `crates-go/auth` — session `Store`, `RequireSession` / `RequireAPIKey` /
  `RequireBearerOrSession` middleware, `SecureEqual`, `VerifyPassword`
  (bcrypt or plaintext), `GenerateSessionToken`, `GenerateShortID`.
- `crates-go/web` — `Render`, `WriteJSON`, `WriteError`, `DecodeJSON`,
  `EmbeddedHandler`, `RedirectWithError`, `RedirectWithSuccess`, `PathInt64`.
- `crates-go/config` — `LoadDotEnv`, `Getenv`, `GetenvInt`, `GetenvBool`.
- `crates-go/sqlite` — `Open(path, schema)` opens SQLite with the project
  defaults (`PRAGMA foreign_keys=ON`, `SetMaxOpenConns(1)`).
- `crates-go/darkmatter` — embedded shared CSS + fonts plus `Mount(mux,
  "/assets")` to register routes on any `*http.ServeMux`.

## Project Structure

```
apps/app-name/
├── main.go              # entry point; loads env, opens DB, builds App, starts server
├── app.go               # App struct + embed.FS + page-data structs
├── routes.go            # *http.ServeMux wiring + middleware
├── db.go                # schema, model, CRUD funcs (or thin wrapper around internal/store)
├── handlers_web.go      # HTML handlers (login, index, CRUD pages)
├── handlers_api.go      # JSON API handlers
├── templates/           # html/template files
│   ├── base.html        # layout with {{ block "title" . }} / {{ block "content" . }}
│   └── *.html           # pages that {{ template "base" . }} or define blocks
├── static/              # CSS, JS, images embedded via embed.FS
├── .env.example
├── Dockerfile           # multi-stage Go build, built from repo root
├── docker-compose.yml   # app-local dev compose
├── go.mod / go.sum
└── README.md
```

Apps with extra surface (e.g. `jotts` has `tui/` + `cmd_*.go` subcommands,
`feeds` has `subscriptions.go` + background poller) layer files on top of this
shape; do not invent new layers unless the app needs them.

## go.mod

Module path: `github.com/stevedylandev/andromeda/apps/<app-name>`. Pin
`go 1.25` (match other apps). Pull the shared crates by their canonical paths
and add `replace` directives for the local checkout:

```go
module github.com/stevedylandev/andromeda/apps/APP_NAME

go 1.25

require (
	github.com/stevedylandev/andromeda/crates-go/auth v0.0.0
	github.com/stevedylandev/andromeda/crates-go/config v0.0.0
	github.com/stevedylandev/andromeda/crates-go/darkmatter v0.0.0
	github.com/stevedylandev/andromeda/crates-go/sqlite v0.0.0
	github.com/stevedylandev/andromeda/crates-go/web v0.0.0
)

replace (
	github.com/stevedylandev/andromeda/crates-go/auth       => ../../crates-go/auth
	github.com/stevedylandev/andromeda/crates-go/config     => ../../crates-go/config
	github.com/stevedylandev/andromeda/crates-go/darkmatter => ../../crates-go/darkmatter
	github.com/stevedylandev/andromeda/crates-go/sqlite     => ../../crates-go/sqlite
	github.com/stevedylandev/andromeda/crates-go/web        => ../../crates-go/web
)
```

Add `crates-go/tui` only if the app ships a TUI. Add `golang.org/x/crypto`,
`yuin/goldmark`, etc. only when the specific app needs them. The Dockerfile
must copy `crates-go/` into the build context for the replace directives.

Do NOT add ORMs, web frameworks (gin, echo, chi), connection pools, or HTTP
client libs unless explicitly requested. The stdlib `net/http` mux (with
method-prefixed patterns like `GET /notes/{short_id}`) is sufficient.

## main.go

Minimal — loads env, sets up logging + DB + sessions, starts the server.

```go
package main

import (
	"html/template"
	"log"
	"log/slog"
	"net/http"
	"os"

	"github.com/stevedylandev/andromeda/crates-go/auth"
	"github.com/stevedylandev/andromeda/crates-go/config"
)

func main() {
	config.LoadDotEnv(".env")
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	dbPath := config.Getenv("APP_DB_PATH", "app.sqlite")
	db, err := openDB(dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	sessions := &auth.Store{
		DB:           db,
		CookieName:   "session",
		CookieSecure: config.GetenvBool("COOKIE_SECURE", false),
	}
	if err := sessions.EnsureSchema(); err != nil {
		log.Fatal(err)
	}
	sessions.PruneExpired()

	tmpl := template.Must(template.ParseFS(appFS, "templates/*.html"))

	password := os.Getenv("APP_PASSWORD")
	if password == "" {
		logger.Warn("APP_PASSWORD not set, using default 'changeme'")
		password = "changeme"
	}
	apiKey := os.Getenv("APP_API_KEY")
	if apiKey == "" {
		logger.Info("APP_API_KEY not set, /api/* will return 403")
	}

	app := &App{
		DB:           db,
		Log:          logger,
		Templates:    tmpl,
		Sessions:     sessions,
		Password:     password,
		APIKey:       apiKey,
		CookieSecure: sessions.CookieSecure,
	}

	addr := config.Getenv("HOST", "127.0.0.1") + ":" + config.Getenv("PORT", "3000")
	logger.Info("APP_NAME server running", "addr", addr)
	if err := http.ListenAndServe(addr, app.routes()); err != nil {
		log.Fatal(err)
	}
}
```

Apps that also expose CLI subcommands (e.g. `jotts`, `sipp`) keep `main.go`
as a dispatcher and move `runServer` into `cmd_server.go`.

## app.go — App struct + embed

Holds dependencies, embedded FS, and per-page data structs.

```go
package main

import (
	"database/sql"
	"embed"
	"html/template"
	"log/slog"

	"github.com/stevedylandev/andromeda/crates-go/auth"
)

//go:embed templates/*.html static/*
var appFS embed.FS

type App struct {
	DB           *sql.DB
	Log          *slog.Logger
	Templates    *template.Template
	Sessions     *auth.Store
	Password     string
	APIKey       string
	CookieSecure bool
}

type indexPageData struct {
	Items []Item
}

type loginPageData struct {
	Error string
}
```

Page-data structs go here too. Keep names like `<page>PageData`.

## routes.go

Wire routes on a `*http.ServeMux` using method-prefixed patterns. Mount
darkmatter assets and static files. Wrap protected routes with
`Sessions.RequireSession` / `auth.RequireAPIKey`.

```go
package main

import (
	"net/http"

	"github.com/stevedylandev/andromeda/crates-go/auth"
	"github.com/stevedylandev/andromeda/crates-go/darkmatter"
	"github.com/stevedylandev/andromeda/crates-go/web"
)

func (a *App) routes() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /static/", web.EmbeddedHandler(appFS, "static"))
	darkmatter.Mount(mux, "/assets")

	requireSession := func(next http.HandlerFunc) http.HandlerFunc {
		return a.Sessions.RequireSession("/login", next)
	}
	requireAPIKey := func(next http.HandlerFunc) http.HandlerFunc {
		return auth.RequireAPIKey(a.APIKey, next)
	}

	mux.HandleFunc("GET /login", a.loginGetHandler)
	mux.HandleFunc("POST /login", a.loginPostHandler)
	mux.HandleFunc("GET /logout", a.logoutHandler)

	mux.HandleFunc("GET /{$}", requireSession(a.indexHandler))
	mux.HandleFunc("GET /items/new", requireSession(a.newItemGetHandler))
	mux.HandleFunc("POST /items", requireSession(a.createItemHandler))
	mux.HandleFunc("GET /items/{short_id}", requireSession(a.viewItemHandler))
	mux.HandleFunc("GET /items/{short_id}/edit", requireSession(a.editItemGetHandler))
	mux.HandleFunc("POST /items/{short_id}", requireSession(a.updateItemHandler))
	mux.HandleFunc("POST /items/{short_id}/delete", requireSession(a.deleteItemHandler))

	mux.HandleFunc("GET /api/items", requireAPIKey(a.apiListItems))
	mux.HandleFunc("POST /api/items", requireAPIKey(a.apiCreateItem))
	mux.HandleFunc("GET /api/items/{short_id}", requireAPIKey(a.apiGetItem))
	mux.HandleFunc("PUT /api/items/{short_id}", requireAPIKey(a.apiUpdateItem))
	mux.HandleFunc("DELETE /api/items/{short_id}", requireAPIKey(a.apiDeleteItem))

	return mux
}
```

Notes:
- `GET /{$}` matches exactly `/` (no prefix matching).
- Path params extracted with `r.PathValue("short_id")`.
- For HTML forms (no JS), delete via `POST /items/{short_id}/delete`. The
  `/api/*` routes use real `DELETE`/`PUT`.

## db.go (or internal/store)

Single-file module: schema, model, CRUD. `crates-go/sqlite.Open` handles the
driver, pragmas, and schema bootstrap.

```go
package main

import (
	"database/sql"
	"errors"

	"github.com/stevedylandev/andromeda/crates-go/auth"
	"github.com/stevedylandev/andromeda/crates-go/sqlite"
)

type Item struct {
	ID        int64  `json:"id"`
	ShortID   string `json:"short_id"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type ItemInput struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

const itemColumns = `id, short_id, title, content, created_at, updated_at`

const schema = `
CREATE TABLE IF NOT EXISTS items (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    short_id   TEXT NOT NULL UNIQUE,
    title      TEXT NOT NULL,
    content    TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);
`

func openDB(path string) (*sql.DB, error) { return sqlite.Open(path, schema) }

func scanItem(s interface{ Scan(...any) error }) (*Item, error) {
	var it Item
	err := s.Scan(&it.ID, &it.ShortID, &it.Title, &it.Content, &it.CreatedAt, &it.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &it, nil
}

func createItem(db *sql.DB, title, content string) (*Item, error) {
	shortID, err := auth.GenerateShortID(10)
	if err != nil {
		return nil, err
	}
	res, err := db.Exec(`INSERT INTO items (short_id, title, content) VALUES (?, ?, ?)`, shortID, title, content)
	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	return scanItem(db.QueryRow(`SELECT `+itemColumns+` FROM items WHERE id = ?`, id))
}

func getItemByShortID(db *sql.DB, shortID string) (*Item, error) {
	return scanItem(db.QueryRow(`SELECT `+itemColumns+` FROM items WHERE short_id = ?`, shortID))
}

func listItems(db *sql.DB) ([]Item, error) {
	rows, err := db.Query(`SELECT ` + itemColumns + ` FROM items ORDER BY id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Item
	for rows.Next() {
		var it Item
		if err := rows.Scan(&it.ID, &it.ShortID, &it.Title, &it.Content, &it.CreatedAt, &it.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

func updateItemByShortID(db *sql.DB, shortID, title, content string) (*Item, error) {
	res, err := db.Exec(`UPDATE items SET title=?, content=?, updated_at=datetime('now') WHERE short_id=?`, title, content, shortID)
	if err != nil {
		return nil, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil, nil
	}
	return getItemByShortID(db, shortID)
}

func deleteItemByShortID(db *sql.DB, shortID string) (bool, error) {
	res, err := db.Exec(`DELETE FROM items WHERE short_id = ?`, shortID)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}
```

Conventions:
- Short IDs via `auth.GenerateShortID(10)`.
- Model fields use `json:"..."` tags so the same struct serializes for the API.
- "Not found" returns `(nil, nil)`, not an error — handlers check `if x == nil`.
- DB path comes from `<APP>_DB_PATH` env (e.g. `JOTTS_DB_PATH`).
- For more complex apps, move this into `internal/store/` and keep `db.go` as
  thin pass-through wrappers (see `apps/jotts`).

## handlers_web.go

HTML handlers. Use `web.Render` for templates and `web.RedirectWithError` for
flash-style errors via query params.

```go
package main

import (
	"net/http"
	"strings"

	"github.com/stevedylandev/andromeda/crates-go/auth"
	"github.com/stevedylandev/andromeda/crates-go/web"
)

func (a *App) loginGetHandler(w http.ResponseWriter, r *http.Request) {
	web.Render(a.Templates, w, "login.html", loginPageData{Error: r.URL.Query().Get("error")}, a.Log)
}

func (a *App) loginPostHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		web.RedirectWithError(w, r, "/login", "Invalid request")
		return
	}
	if !auth.SecureEqual(r.FormValue("password"), a.Password) {
		web.RedirectWithError(w, r, "/login", "Invalid password")
		return
	}
	token, err := a.Sessions.Create()
	if err != nil {
		a.Log.Error("create session failed", "err", err)
		web.RedirectWithError(w, r, "/login", "Server error")
		return
	}
	http.SetCookie(w, a.Sessions.SessionCookie(token))
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (a *App) logoutHandler(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(a.Sessions.CookieName); err == nil && c.Value != "" {
		a.Sessions.Delete(c.Value)
	}
	http.SetCookie(w, a.Sessions.ClearCookie())
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (a *App) createItemHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		web.RedirectWithError(w, r, "/items/new", "Invalid request")
		return
	}
	title := strings.TrimSpace(r.FormValue("title"))
	if title == "" {
		web.RedirectWithError(w, r, "/items/new", "Title is required")
		return
	}
	it, err := createItem(a.DB, title, r.FormValue("content"))
	if err != nil {
		a.Log.Error("create item failed", "err", err)
		web.RedirectWithError(w, r, "/items/new", "Failed to create item")
		return
	}
	http.Redirect(w, r, "/items/"+it.ShortID, http.StatusSeeOther)
}
```

Patterns:
- Always `auth.SecureEqual` for password compare (constant-time).
- Use `http.StatusSeeOther` (303) for POST-redirect-GET.
- Pre-rendered HTML (e.g. markdown) goes through `template.HTML` in the
  page-data struct, not via `|safe`-style filters.

## handlers_api.go

JSON API handlers. Use `web.WriteJSON`, `web.WriteError`, `web.DecodeJSON`.

```go
package main

import (
	"net/http"

	"github.com/stevedylandev/andromeda/crates-go/web"
)

func (a *App) apiListItems(w http.ResponseWriter, r *http.Request) {
	items, err := listItems(a.DB)
	if err != nil {
		a.Log.Error("list items failed", "err", err)
		web.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}
	web.WriteJSON(w, http.StatusOK, items)
}

func (a *App) apiCreateItem(w http.ResponseWriter, r *http.Request) {
	var in ItemInput
	if !web.DecodeJSON(w, r, &in) {
		return
	}
	if in.Title == "" {
		web.WriteError(w, http.StatusBadRequest, "title is required")
		return
	}
	it, err := createItem(a.DB, in.Title, in.Content)
	if err != nil {
		a.Log.Error("create item failed", "err", err)
		web.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}
	web.WriteJSON(w, http.StatusCreated, it)
}

func (a *App) apiGetItem(w http.ResponseWriter, r *http.Request) {
	it, err := getItemByShortID(a.DB, r.PathValue("short_id"))
	if err != nil {
		web.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if it == nil {
		web.WriteError(w, http.StatusNotFound, "not found")
		return
	}
	web.WriteJSON(w, http.StatusOK, it)
}
```

## Authentication

`crates-go/auth` provides three middleware patterns. Pick based on app shape:

### Session/cookie auth (web-facing apps)

Use `*auth.Store` for password-login web apps (jotts, feeds, bookmarks, etc).
Sessions stored in the same SQLite DB via `EnsureSchema()`. Routes guarded
with `store.RequireSession("/login", handler)`.

Key API:
- `store.Create() (token, err)` — issue new session row.
- `store.SessionCookie(token) *http.Cookie` — `HttpOnly`, `SameSite=Strict`,
  7-day MaxAge.
- `store.ClearCookie()` — `MaxAge=-1` cookie for logout.
- `store.HasValid(r)` — check the incoming request's cookie.
- `store.PruneExpired()` — call at boot to GC stale rows.

### API key auth (header-based)

`auth.RequireAPIKey(expectedKey, next)`. Reads `X-API-Key` header,
`SecureEqual` compare. Returns 403 if `expectedKey` is empty (auth disabled by
config), 401 on mismatch. Use this for `/api/*` routes when there's also a
session-cookie web UI.

### Bearer-or-Session (mixed surfaces)

`auth.RequireBearerOrSession(store, expectedBearer, next)` — accepts either
`Authorization: Bearer <token>` or a valid session cookie. Used by apps that
expose the same endpoint to a CLI client AND a logged-in browser user (e.g.
`sipp`, `jotts upload`).

### Env vars

| Variable | Purpose | Default |
|----------|---------|---------|
| `HOST` | bind address | `127.0.0.1` (set `0.0.0.0` in Docker) |
| `PORT` | listen port | `3000` |
| `<APP>_DB_PATH` | SQLite path | `<app>.sqlite` |
| `<APP>_PASSWORD` | session login password | none → "changeme" warning |
| `<APP>_API_KEY` | API key for `/api/*` | none → 403 |
| `COOKIE_SECURE` | HTTPS-only cookie flag | `false` |

App-specific env vars are prefixed with the uppercase app name
(`JOTTS_PASSWORD`, `FEEDS_API_KEY`). `HOST`, `PORT`, `COOKIE_SECURE` are not
prefixed.

## Templates (html/template)

Templates live in `templates/`, parsed once at startup via
`template.ParseFS(appFS, "templates/*.html")` and rendered with
`web.Render(t, w, "name.html", data, log)`.

Use `{{ define "..." }}` / `{{ template "..." . }}` for layout composition.
The base file defines block names; pages define the content blocks.

**templates/base.html:**

```html
{{ define "base" }}
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>{{ template "title" . }}</title>
  <meta name="theme-color" content="#121113" />
  <link rel="stylesheet" href="/assets/darkmatter.css">
  <link rel="stylesheet" href="/static/styles.css">
</head>
<body>
  <div class="container">
    {{ template "content" . }}
  </div>
</body>
</html>
{{ end }}
```

**templates/index.html:**

```html
{{ define "title" }}Items{{ end }}
{{ define "content" }}
  {{ range .Items }}
    <a href="/items/{{ .ShortID }}">{{ .Title }}</a>
  {{ end }}
{{ end }}
{{ template "base" . }}
```

Notes:
- Link CSS via `/static/styles.css` (app-local) and `/assets/darkmatter.css`
  (shared theme served by `darkmatter.Mount`).
- Forms POST to web routes, not `/api/*`.
- Pass pre-rendered HTML as `template.HTML(...)`; `html/template` auto-escapes
  everything else.
- Flash errors via `?error=...` query param + `web.RedirectWithError` /
  `RedirectWithSuccess`.

## Static assets

Embedded via `//go:embed templates/*.html static/*` on a single `appFS`.
Serve under `/static/` via `web.EmbeddedHandler(appFS, "static")`. Shared
theme files (CSS + fonts) come from `darkmatter.Mount(mux, "/assets")`.

## Logging (slog)

Standard library `log/slog`. Build once in `main`:

```go
logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
```

Pass it into `App` and pass it to `web.Render` (it logs template errors).
Use `logger.Error("msg", "err", err)`, `logger.Warn(...)`, `logger.Info(...)`.

## Dockerfile

Multi-stage, built from the repo root so `crates-go/` is available for the
`replace` directives. Pure Go (`CGO_ENABLED=0`) because the SQLite driver is
`modernc.org/sqlite`.

```dockerfile
# Build from repo root: docker build -t APP_NAME -f apps/APP_NAME/Dockerfile .
FROM golang:1.25-bookworm AS builder
WORKDIR /app
COPY crates-go/ ./crates-go/
COPY apps/APP_NAME/go.mod apps/APP_NAME/go.sum ./apps/APP_NAME/
WORKDIR /app/apps/APP_NAME
RUN go mod download
COPY apps/APP_NAME/ ./
RUN CGO_ENABLED=0 go build -o /APP_NAME .

FROM debian:bookworm-slim
COPY --from=builder /APP_NAME /usr/local/bin/APP_NAME
WORKDIR /data
ENV HOST=0.0.0.0
ENV PORT=3000
EXPOSE 3000
CMD ["APP_NAME"]
```

If the app makes outbound HTTPS requests, add to the final stage:
```dockerfile
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates && rm -rf /var/lib/apt/lists/*
```

If the app has CLI subcommands (jotts), set `CMD ["APP_NAME", "server"]`.

## App-local docker-compose.yml

```yaml
services:
  app:
    build:
      context: ../..
      dockerfile: apps/APP_NAME/Dockerfile
    ports:
      - "${PORT:-3000}:${PORT:-3000}"
    environment:
      - APP_NAME_PASSWORD=${APP_NAME_PASSWORD:-changeme}
      - APP_NAME_DB_PATH=/data/APP_NAME.sqlite
      - COOKIE_SECURE=false
      - HOST=0.0.0.0
      - PORT=${PORT:-3000}
    volumes:
      - APP_NAME-data:/data
    restart: unless-stopped

volumes:
  APP_NAME-data:
```

## .env.example

Always ship one listing every var the app reads, with short comments.

## Wiring up a new app

A new app is not done when `go run .` works. Several workspace-level files
list every app explicitly and must be updated, or CI / Docker images / deploy
break silently.

### 1. Root `docker-compose.yml` (production)

Pulls images from GHCR. Add a service + named volume:

```yaml
services:
  APP_NAME:
    image: ghcr.io/stevedylandev/andromeda/APP_NAME:latest
    restart: unless-stopped
    ports:
      - "PORT:3000"
    volumes:
      - APP_NAME_data:/data
    env_file: apps/APP_NAME/.env
```

Pick a unique host port (existing: 3000 sipp, 3030 jotts, 3456 og, 3535 posts,
3565 shrink, 4555 feeds, 4646 library, 6754 cellar). Drop the `volumes` /
`env_file` lines if stateless (see `shrink`, `og`).

If the app holds data worth backing up, mount its volume read-only into the
`backup` service.

### 2. `.github/workflows/docker.yml`

Two spots — both list every app:

```yaml
ALL='["backup","bookmarks","cellar","easel","feeds","jotts","library","og","posts","shrink","sipp","APP_NAME"]'
```

```yaml
for app in backup bookmarks cellar easel feeds jotts library og posts shrink sipp APP_NAME; do
```

### 3. `.github/workflows/docker-test.yml`

Same two lists — keep in lockstep with `docker.yml`.

### 4. App-local `docker-compose.yml`

Per-app dev compose (see above).

## Useful commands

Per-app (from `apps/<app>/`):

```bash
go mod tidy
go build ./...
go vet ./...
go test ./...
go run .                # or: go run . server  for apps with subcommands
```

Repo-wide (from root):

```bash
make go-check                       # fmt + test + vet across all Go modules
make go-test
make go-vet
make go-fmt
make go-app-test APP=APP_NAME
make go-app-vet  APP=APP_NAME
make go-app-fmt  APP=APP_NAME
```

The shared crates (`crates-go/web`, `crates-go/auth`, `crates-go/config`,
`crates-go/sqlite`, `crates-go/darkmatter`) are each their own module — run
`go build ./...` inside any to check in isolation.

## Checklist

When scaffolding a new app:

1. `mkdir apps/APP_NAME && cd apps/APP_NAME && go mod init github.com/stevedylandev/andromeda/apps/APP_NAME`
2. Add shared-crate `require` + `replace` directives to `go.mod` (auth, config, sqlite, web, darkmatter).
3. `main.go` — env load, DB open, sessions bootstrap, `App` build, `ListenAndServe`.
4. `app.go` — `App` struct, `//go:embed`, page-data structs.
5. `routes.go` — mux wiring, middleware, `/static/` + `darkmatter.Mount`.
6. `db.go` — schema, model, CRUD via `crates-go/sqlite.Open`.
7. `handlers_web.go` — login/logout + HTML CRUD pages.
8. `handlers_api.go` — JSON CRUD endpoints behind `RequireAPIKey`.
9. `templates/base.html` + per-page templates using `{{ define }}` blocks.
10. `static/styles.css` + any app-specific assets.
11. `.env.example` listing every env var.
12. `Dockerfile` (multi-stage, `CGO_ENABLED=0`) + app-local `docker-compose.yml`.
13. Add service + volume to root `docker-compose.yml` (+ `backup` mount if stateful).
14. Add app name to both `ALL` array + `for app in ...` loops in
    `.github/workflows/docker.yml` AND `.github/workflows/docker-test.yml`.
15. `go mod tidy`, then `make go-app-vet APP=APP_NAME` and `make go-app-test APP=APP_NAME`.
16. `docker build -f apps/APP_NAME/Dockerfile .` from repo root to verify image.

## What NOT to include

- No web frameworks (gin, echo, chi, fiber) — stdlib `net/http` mux is enough.
- No ORMs — raw `database/sql` + `?` placeholders.
- No connection pools — `sqlite.Open` sets `MaxOpenConns(1)` on purpose.
- No cgo SQLite drivers — use `modernc.org/sqlite` via `crates-go/sqlite`.
- No external CSS frameworks unless specified — use `crates-go/darkmatter`.
- No JS frontend / build step — `html/template` rendered server-side.
- No CLI / TUI deps (cobra, bubbletea) unless the app actually needs them.
- No `replace` directives for anything outside `crates-go/`.
