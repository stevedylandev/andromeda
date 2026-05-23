# easel

Daily painting from the Art Institute of Chicago, persisted to SQLite. Past
days browsable; future days unavailable.

## Install

**Homebrew:**

```bash
brew install stevedylandev/tap/easel
```

**Curl install (Linux/macOS):**

```bash
curl -fsSL https://raw.githubusercontent.com/stevedylandev/andromeda/main/install.sh | sh -s -- easel
```

**Prebuilt binary:** Grab the right archive from the [releases page](https://github.com/stevedylandev/andromeda/releases?q=easel%2F) and drop the binary somewhere on your `$PATH`.

**From source:**

```bash
git clone https://github.com/stevedylandev/andromeda
cd andromeda/apps/easel
go build .
```

## Routes

- `GET /` — today's artwork
- `GET /day/{YYYY-MM-DD}` — specific past day
- `GET /archive` — full archive
- `GET /api/today` / `GET /api/day/{date}` / `GET /api/archive` — JSON
- `GET /feed.xml` — Atom feed

## Env

See `.env.example`. Notes: timezone uses Go's `time.LoadLocation`, which needs
the system tzdata (Debian slim base in the Dockerfile pulls `tzdata`).
