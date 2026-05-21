# cellar

Wine tasting log with optional Anthropic vision (label analysis) and per-wine
RSS feed.

## Notes

- Anthropic `/v1/messages` called via stdlib `net/http` (no SDK).
- Image processing uses stdlib `image` decode + JPEG re-encode at quality 75.
  EXIF orientation is not respected; rotate before upload if needed.
- Multipart upload limit 10 MB.

See `.env.example` for config.
