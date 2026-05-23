# posts

CMS blog with admin, pages, file uploads, markdown rendering, RSS, zip
import/export.

## Install

**Homebrew:**

```bash
brew install stevedylandev/tap/posts
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

See `.env.example`.
