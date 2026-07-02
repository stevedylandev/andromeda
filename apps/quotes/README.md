# quotes

A minimal quote-a-day site. The landing page shows a single **quote of the day**
from classic literature — white, centered, on a dark background. `/admin` is a
session-protected page to add and manage quotes, matching the other andromeda
apps.

## Run

```sh
cp .env.example .env   # set QUOTES_PASSWORD
go run .
```

Landing: `http://localhost:3000/` — the quote of the day (deterministic, rotates
at UTC midnight).
Admin: `http://localhost:3000/admin` — log in with `QUOTES_PASSWORD`.

## Seeding from a CSV

`quotes.csv` is a Goodreads-style export (`quote,author,category`). The seed
command imports only classic-literature quotes, matched case-sensitively against
the author/title list in `classic_authors.txt`:

```sh
go run . seed quotes.csv
```

- Edit `classic_authors.txt` (one author or book title per line, `#` comments
  allowed) to widen or narrow the selection, then re-run the command.
- Attribution is split at the first comma: `author` and `source` (book title).
- Re-seeding is idempotent — quotes already present (matched on text + author)
  are skipped.
- The author list is also **embedded in the binary**, so `seed` works even when
  `classic_authors.txt` is not on disk (e.g. inside a container). An on-disk
  file always takes precedence, so local edits apply without a rebuild.

### Seeding a Docker volume

The image only ships the binary — the DB lives on the `quotes_data` volume
(`QUOTES_DB_PATH=/data/quotes.sqlite`), and the 138 MB `quotes.csv` is
deliberately excluded from the build (see `.dockerignore`). So seeding is a
one-off `run` that bind-mounts the CSV and writes into the same named volume the
service uses:

```sh
# From the repo root (uses the `quotes` service's volume + env)
docker compose run --rm \
  -v "$PWD/apps/quotes/quotes.csv:/seed/quotes.csv:ro" \
  quotes quotes seed /seed/quotes.csv

# Then start the service normally
docker compose up -d quotes
```

The author list comes from the embedded copy. To seed with a different list
without rebuilding the image, mount your edited file over the working directory
(`/data`) too:

```sh
docker compose run --rm \
  -v "$PWD/apps/quotes/quotes.csv:/seed/quotes.csv:ro" \
  -v "$PWD/apps/quotes/classic_authors.txt:/data/classic_authors.txt:ro" \
  quotes quotes seed /seed/quotes.csv
```

Because seeding is idempotent, you can re-run either command after widening the
list to pull in the newly matched quotes.

## API (public, read-only)

- `GET /api/quotes?limit=100` — most recent quotes
- `GET /api/quotes/today` — the quote of the day
- `GET /api/quotes/{short_id}` — a single quote

## Environment

| Var | Default | Notes |
| --- | --- | --- |
| `QUOTES_PASSWORD` | _(empty)_ | Admin login password (plaintext or bcrypt hash). Empty disables login. |
| `QUOTES_API_KEY` | _(empty)_ | Reserved; the read API is currently public. |
| `QUOTES_DB_PATH` | `quotes.sqlite` | SQLite file path. |
| `HOST` / `PORT` | `0.0.0.0` / `3000` | Listen address. |
| `BASE_URL` | `http://localhost:3000` | Used in social meta tags. |
| `COOKIE_SECURE` | `false` | Set `true` behind HTTPS. |
