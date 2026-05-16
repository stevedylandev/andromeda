# cellar-go

Go rewrite of [cellar](../cellar). Wine tasting log with optional Anthropic
vision (label analysis) and per-wine RSS feed.

## Notes vs Rust version

- Anthropic `/v1/messages` called via stdlib `net/http` (no SDK).
- Image processing uses stdlib `image` decode + JPEG re-encode at quality 75.
  EXIF orientation is not respected; rotate before upload if needed.
- Multipart upload limit kept at 10 MB.

See `.env.example` for config.
