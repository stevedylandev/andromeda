# og-go

Go rewrite of [og](../og). Open Graph tag inspector for any URL.

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
- `GET /assets/darkmatter.css` + `/assets/fonts/*` — served by `crates-go/darkmatter`

## Build

```bash
go build .
```

The binary embeds all templates and static assets.
