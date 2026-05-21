# posts

CMS blog with admin, pages, file uploads, markdown rendering, RSS, zip
import/export.

## Notes

- Upload storage supports local filesystem (`UPLOADS_DIR`, default `uploads`)
  or Cloudflare R2 when `R2_BUCKET` and credentials are set.
- Markdown: `github.com/yuin/goldmark` with GFM + Footnotes.
- Zip via stdlib `archive/zip`. Upload limit 10 MB; import zip limit 50 MB.
- API: `GET /api/posts` and `GET /api/posts/{slug}` (permissive CORS).

See `.env.example`.
