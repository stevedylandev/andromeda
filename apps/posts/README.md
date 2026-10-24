# posts

CMS blog with admin, pages, file uploads, markdown rendering, RSS, zip
import/export.

## Install

**Homebrew:**

```bash
brew install stevedylandev/tap/posts
```

**Curl install (Linux/macOS):**

```bash
curl -fsSL https://raw.githubusercontent.com/stevedylandev/andromeda/main/install.sh | sh -s -- posts
```

**PowerShell (Windows):**

```powershell
& ([scriptblock]::Create((irm https://raw.githubusercontent.com/stevedylandev/andromeda/main/install.ps1))) posts
```

**Prebuilt binary:** Grab the right archive from the [releases page](https://github.com/stevedylandev/andromeda/releases?q=posts%2F) and drop the binary somewhere on your `$PATH`.

**From source:**

```bash
git clone https://github.com/stevedylandev/andromeda
cd andromeda/apps/posts
go build .
```

## Notes

- Upload storage supports local filesystem (`UPLOADS_DIR`, default `uploads`)
  or Cloudflare R2 when `R2_BUCKET` and credentials are set.
- Markdown: `github.com/yuin/goldmark` with GFM + Footnotes.
- Zip via stdlib `archive/zip`. Upload limit 10 MB; import zip limit 50 MB.
- API: `GET /api/posts` and `GET /api/posts/{slug}` (permissive CORS).
- `POST /api/posts` creates a post (requires `X-API-Key` header matching
  `POSTS_API_KEY`; disabled when unset). JSON body: `content` (required),
  optional `title`, `slug`, `status` (`draft`|`published`, default `draft`),
  `alias`, `canonical_url`, `published_date`, `meta_description`,
  `meta_image`, `lang`, `tags`, `weather`.
- `POST /api/files` uploads a file (same `X-API-Key` auth). Multipart form:
  `file` (required, 10 MB max) and optional `name`, which becomes the stored
  filename (slugified, extension kept or taken from the upload; 409 if taken).
  Without `name` a random ID is used. Returns 201 with JSON `url`,
  `filename`, `original_name`, `content_type`, `size`.

- Public pages are [templ](https://templ.guide) components (`*.templ`). After
  editing them run `go generate ./...` and commit the generated `*_templ.go`.
- `/admin/templates` lets you replace the body (inside `<main>`) of the home,
  post, page and posts list views with a Go `html/template`. The editor starts
  from `starters/*.html`, which must stay in sync with the templ defaults
  (`TestStartersMatchTemplDefaults` checks this). Templates are validated
  against sample data on save; if one fails at request time the default is
  rendered instead.

See `.env.example`.
