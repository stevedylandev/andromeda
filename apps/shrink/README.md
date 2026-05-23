# shrink

JPEG compression + resize via stdlib `image` plus `golang.org/x/image/draw`
for Catmull-Rom scaling.

## Install

**Homebrew:**

```bash
brew install stevedylandev/tap/shrink
```

**Curl install (Linux/macOS):**

```bash
curl -fsSL https://raw.githubusercontent.com/stevedylandev/andromeda/main/install.sh | sh -s -- shrink
```

**Prebuilt binary:** Grab the right archive from the [releases page](https://github.com/stevedylandev/andromeda/releases?q=shrink%2F) and drop the binary somewhere on your `$PATH`.

**From source:**

```bash
git clone https://github.com/stevedylandev/andromeda
cd andromeda/apps/shrink
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
| `HOST` | `127.0.0.1` | Bind host |
| `PORT` | `3000` | Server port |

## Routes

- `GET /` — upload UI
- `POST /compress` — multipart upload (`file`, `quality` 1-100, optional `width`)
- `GET /static/*` — embedded assets
- `/assets/*` — darkmatter css/fonts

## Notes

JPEG EXIF metadata is preserved after recompression, with GPS data stripped.
Upload limit 20 MB.
