# Sipp

Minimal code sharing

https://github.com/user-attachments/assets/cadafb70-f796-456d-bfd9-e88704e7132c

## Features

- Single binary for web server and TUI
- Create snippets and share on the web
- Raw output for CLI tools — `curl`, `wget`, and `httpie` get plain text automatically
- Interactive TUI with authenticated access for snippet management
- Minimal, fast, and low memory consumption

## Quickstart

**1. Install**

Install via the [releases](https://github.com/stevedylandev/sipp/releases) page, or directly with `cargo`

```bash
cargo install sipp-so
```

To confirm it was installed correctly run the following

```bash
sipp --help
```

**2. Start Server**

For demo purposes you can run this locally, but ideally this would be run in a deployment server with a proper ENV setup with your admin key.

```bash
sipp server --port 3000
```

**3. Create a Snippet**

You can either open up `http://localhost:3000` and create a snippet in a web browser, or use the TUI. In the same directory, open a new terminal window and use 

```bash
# Path to file
sipp path/to/file.rs

# Or use the interactive tui 
sipp
```

## Demo Instance

A small instance running at [sipp.so](https://sipp.so) that can be used for testing and demo purposes.

```bash
sipp -r https://sipp.so
```

>[!WARNING]
>All snippets created here are public and might be deleted at any time; host your own instance with your own API key for personal use!

## Install

Sipp can be installed several ways

### Releases

Visit the [releases](https://github.com/stevedylandev/sipp/releases) page and install through cURL script and other methods.

### Homebrew

```
brew install stevedylandev/tap/sipp-so
```

### Cargo

```bash
cargo install sipp-so
```

## Usage

### CLI

```
sipp [OPTIONS] [FILE] [COMMAND]
```

#### Commands

| Command | Description |
|---|---|
| `server` | Start the web server |
| `tui` | Launch the interactive TUI |
| `auth` | Save remote URL and API key to config file |

#### Arguments

| Argument | Description |
|---|---|
| `[FILE]` | File path to create a snippet from |

#### Options

| Option | Description |
|---|---|
| `-r, --remote <URL>` | Remote server URL (e.g. `http://localhost:3000`) (env: `SIPP_REMOTE_URL`) |
| `-k, --api-key <KEY>` | API key for authenticated operations (env: `SIPP_API_KEY`) |

### Server

Sipp includes a built-in web server powered by Axum. Start it with:

```bash
sipp server --port 3000 --host localhost
```

#### Environment Variables

| Variable | Description |
|---|---|
| `SIPP_API_KEY` | API key for protecting endpoints |
| `SIPP_AUTH_ENDPOINTS` | Comma-separated list of endpoints requiring auth: `api_list`, `api_create`, `api_get`, `api_delete`, `all`, or `none` (defaults to `api_delete,api_list`) |
| `SIPP_MAX_CONTENT_SIZE` | Maximum snippet content size in bytes (defaults to `512000` / 500 KB) |
| `SIPP_DB_PATH` | Custom path for the SQLite database file (defaults to `sipp.sqlite` in the working directory) |

The server stores snippets in a local `sipp.sqlite` SQLite database.

#### API Endpoints

| Method | Endpoint | Description |
|---|---|---|
| `GET` | `/api/snippets` | List all snippets |
| `POST` | `/api/snippets` | Create a snippet (`{"name": "...", "content": "..."}`) |
| `GET` | `/api/snippets/{short_id}` | Get a snippet by ID |
| `PUT` | `/api/snippets/{short_id}` | Update a snippet (`{"name": "...", "content": "..."}`) |
| `DELETE` | `/api/snippets/{short_id}` | Delete a snippet by ID |

Authenticated endpoints require an `x-api-key` header.

#### Raw Output for CLI Tools

When you access a snippet URL (`/s/{short_id}`) with `curl`, `wget`, or `httpie`, the server returns the raw content as plain text instead of HTML:

```bash
curl https://sipp.so/s/abc123
```

### TUI

The Sipp TUI makes it easy to create, copy, share, and manage your snippets either locally or remotely. Launch it with:

```bash
# Launch TUI (default behavior when no file argument is given)
sipp

# Or explicitly
sipp tui

# With remote options
sipp -r https://sipp.so -k your-api-key
```

#### Local Access

If you are running `sipp` in the same directory as the `sipp.sqlite` file created by the server instance, the TUI will automatically access the datebase locally and you can edit it directly.

#### Remote Access

To access a remote instance of Sipp make sure to do the following:
- Set the `SIPP_API_KEY` variable in your server instance
- Run `sipp auth` to enter in your server instance URL and the API key, which will be stored under `$HOME/.config/sipp`. You can also set these with the ENV variables `SIPP_REMOTE_URL` and `SIPP_API_KEY`

>[!NOTE]
>You can try a limited remote instance without an API key with `sipp -r https://sipp.so`

#### Actions

While inside the TUI the following actions are available

| Key | Action |
|---|---|
| `j`/`↓` | Move down / Scroll down |
| `k`/`↑` | Move up / Scroll up |
| `Enter` | Focus content pane |
| `Esc` | Back / Quit |
| `y` | Copy snippet content |
| `Y` | Copy snippet link |
| `o` | Open in browser |
| `e` | Edit snippet |
| `d` | Delete snippet |
| `c` | Create snippet |
| `/` | Search snippets |
| `r` | Refresh snippets (remote only) |
| `q` | Quit |
| `?` | Toggle help |

## Deployment

Since Sipp is a single binary it can be run in virtually any enviornment.

### Railway

[![Deploy on Railway](https://railway.com/button.svg)](https://railway.com/deploy/Axcf_D?referralCode=JGcIp6)

### Docker

A `Dockerfile` and `docker-compose.yml` are included in the repository.

```bash
# Using Docker Compose (recommended)
SIPP_API_KEY=your-secret-key docker compose up -d

# Or build and run manually
docker build -t sipp .
docker run -p 3000:3000 -e SIPP_API_KEY=your-secret-key -v sipp-data:/data sipp
```

