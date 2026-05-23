# og

Open Graph tag inspector for any URL.

## Install

**Homebrew:**

```bash
brew install stevedylandev/tap/og
```

**Prebuilt binary:** Grab the right archive from the [releases page](https://github.com/stevedylandev/andromeda/releases?q=og%2F) and drop the binary somewhere on your `$PATH`.

**From source:**

```bash
git clone https://github.com/stevedylandev/andromeda
cd andromeda/apps/og
go build .
```

## Quickstart

```bash
cp .env.example .env
go run .
```

Then open `http://localhost:3000`.

### Environment Variables

| Variable | Default | Description |
|---|---|---|
| `HOST` | `0.0.0.0` | Bind host |
| `PORT` | `3000` | Server port |

## Routes

- `GET /` — search form
- `POST /check` — inspect a URL (form field: `url`)
- `GET /static/*` — embedded favicon, styles, etc.
- `GET /assets/darkmatter.css` + `/assets/fonts/*` — served by `pkg/darkmatter`

## Build

```bash
go build .
```

The binary embeds all templates and static assets.
