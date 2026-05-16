# sipp-go

Go rewrite of [sipp](../sipp). Two binaries:

- `cmd/server` — web server only (HTTP + admin + API + syntax highlight via
  `github.com/alecthomas/chroma/v2`).
- `cmd/sipp` — CLI dispatcher: `sipp server`, or `sipp <file>` to upload a
  snippet to a remote instance via the JSON API.

## Notes vs Rust version

- **Interactive TUI not ported.** The Rust binary uses `ratatui` +
  `crossterm`; build with the Rust version if you need it.
- Syntax highlighting uses Chroma (replaces syntect). The darkmatter
  `.tmTheme` is not reused; Chroma's `monokai` style ships by default.
- Snippet schema and routes match the Rust app; existing SQLite files are
  compatible.

## Quickstart

```bash
cp .env.example .env
go run ./cmd/server
# or
go run ./cmd/sipp server --port 3000
```

Upload a file:

```bash
SIPP_REMOTE_URL=http://localhost:3000 SIPP_API_KEY=$KEY \
  go run ./cmd/sipp ./path/to/file.go
```

See `.env.example` for env vars.
