# sipp-go

Go rewrite of [sipp](../sipp). Single binary with subcommands:

- `sipp server [--host H] [--port P]` — web server (HTTP + admin + API +
  syntax highlight via `github.com/alecthomas/chroma/v2`).
- `sipp tui` — interactive TUI.
- `sipp auth` — save remote URL + API key to config.
- `sipp <file>` — upload a snippet to a remote instance via the JSON API.

## Notes vs Rust version

- TUI uses Bubble Tea (Rust uses `ratatui` + `crossterm`).
- Syntax highlighting uses Chroma (replaces syntect). The darkmatter
  `.tmTheme` is not reused; Chroma's `monokai` style ships by default.
- Snippet schema and routes match the Rust app; existing SQLite files are
  compatible.

## Quickstart

```bash
cp .env.example .env
go run . server --port 3000
```

Upload a file:

```bash
SIPP_REMOTE_URL=http://localhost:3000 SIPP_API_KEY=$KEY \
  go run . ./path/to/file.go
```

See `.env.example` for env vars.
