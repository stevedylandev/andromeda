# cellar

Wine tasting log with optional Anthropic vision (label analysis) and per-wine
RSS feed.

## Install

**Homebrew:**

```bash
brew install stevedylandev/tap/cellar
```

**Curl install (Linux/macOS):**

```bash
curl -fsSL https://raw.githubusercontent.com/stevedylandev/andromeda/main/install.sh | sh -s -- cellar
```

**Prebuilt binary:** Grab the right archive from the [releases page](https://github.com/stevedylandev/andromeda/releases?q=cellar%2F) and drop the binary somewhere on your `$PATH`.

**From source:**

```bash
git clone https://github.com/stevedylandev/andromeda
cd andromeda/apps/cellar
go build .
```

## Notes

- Anthropic `/v1/messages` called via stdlib `net/http` (no SDK).
- Image processing uses stdlib `image` decode + JPEG re-encode at quality 75.
  EXIF orientation is not respected; rotate before upload if needed.
- Multipart upload limit 10 MB.

See `.env.example` for config.
