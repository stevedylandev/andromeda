# posts-go

Go rewrite of [posts](../posts). CMS blog with admin, pages, file uploads,
markdown rendering, RSS, zip import/export.

## Notes vs Rust version

- **R2/S3 storage dropped.** Local filesystem only (`UPLOADS_DIR`, default
  `uploads`). The `files.storage_backend` column stays for schema parity but
  is always `"local"`.
- Markdown: `github.com/yuin/goldmark` with GFM + Footnotes (replaces
  pulldown-cmark).
- Zip via stdlib `archive/zip`. Upload limit 10 MB; import zip limit 50 MB.
- API: `GET /api/posts` and `GET /api/posts/{slug}` (permissive CORS).

See `.env.example`.
