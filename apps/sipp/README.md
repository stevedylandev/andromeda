# sipp

Code sharing. Single binary with subcommands:

- `sipp server [--host H] [--port P]` — web server (HTTP + admin + API +
  syntax highlight via `github.com/alecthomas/chroma/v2`).
- `sipp tui` — interactive Bubble Tea TUI.
- `sipp auth` — save remote URL + API key to config.
- `sipp <file>` — upload a snippet to a remote instance via the JSON API.

## Notes

- Syntax highlighting uses Chroma with the `monokai` style by default.

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
