# Shrink

![cover](https://files.stevedylan.dev/shrink-demo.png)

A minimal image compression app

## Quickstart

```bash
git clone https://github.com/stevedylandev/shrink.git
cd shrink
cargo build --release
./target/release/shrink
```

### Environment Variables

| Variable | Description | Default |
|---|---|---|
| `HOST` | Server bind host | `127.0.0.1` |
| `PORT` | Server bind port | `3000` |

## Overview

A simple self-hosted tool for compressing and resizing images. Upload an image, set your desired quality and optional width, and download the compressed JPEG. A few highlights:

- Single Rust binary
- Compress images to JPEG with configurable quality (1-100)
- Optional resize by width (preserves aspect ratio)
- 20MB upload limit

## Structure

```
shrink/
├── src/
│   ├── main.rs        # Entry point and server startup
│   └── server.rs      # Axum routes and image compression logic
├── templates/
│   └── index.html     # Upload UI
├── static/            # Fonts and static assets
├── Dockerfile
└── docker-compose.yml
```

## Deployment

### Docker (recommended)

```bash
git clone https://github.com/stevedylandev/shrink.git
cd shrink
docker compose up -d
```

This will start Shrink on port `3000`.

### Binary

```bash
cargo build --release
```

The resulting binary at `./target/release/shrink` is self-contained. Copy it to your server along with the `static/` and `templates/` directories, then run it directly.

## License

[MIT](LICENSE)
