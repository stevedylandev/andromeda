# habbits

A minimal, single-admin habit tracker. Define custom habits (each with a typed
value) and log records against them over time. The whole app is behind a
session login — there are no public pages.

## Concepts

- **Habit** — something you track. Has a `name`, a `value_type`
  (`int` | `float` | `bool` | `string`), and optional `unit` and `description`.
- **Record** — a single data point for a habit: a `value` (validated against the
  habit's type) and a `recorded_at` timestamp. The timestamp auto-fills to the
  current time on entry but can be edited/backdated in the UI.

## Routes

Web (all require a session except the login flow):

| Method | Path | Description |
| ------ | ---- | ----------- |
| GET  | `/login` / `/logout`            | Session login / logout |
| GET  | `/`                             | Dashboard: create habit, add record, recent activity |
| POST | `/habits`                       | Create a habit |
| GET  | `/habits/{short_id}`            | Habit detail: edit habit + its records |
| POST | `/habits/{short_id}`            | Update a habit |
| POST | `/habits/{short_id}/delete`     | Delete a habit (cascades its records) |
| POST | `/records`                      | Create a record |
| POST | `/records/{short_id}`           | Update a record |
| POST | `/records/{short_id}/delete`    | Delete a record |

Read-only JSON API (requires `X-API-Key` header; disabled when `HABBITS_API_KEY`
is empty):

| Method | Path | Description |
| ------ | ---- | ----------- |
| GET | `/api/habits`            | All habits |
| GET | `/api/records?limit=100` | Recent records (max 500) |

## Configuration

Copy the values below into `apps/habbits/.env` (create the file locally; it is
git-ignored):

```
HABBITS_PASSWORD=changeme        # admin password (plaintext or bcrypt hash); empty disables login
HABBITS_API_KEY=                 # optional; enables the read-only JSON API when set
HABBITS_DB_PATH=habbits.sqlite   # SQLite file path
HOST=0.0.0.0
PORT=3000
BASE_URL=http://localhost:3000
COOKIE_SECURE=false              # set true behind HTTPS
```

## Development

```
cd apps/habbits
go mod tidy
go run .          # serves on $HOST:$PORT
```

Or with Docker (built from repo root):

```
docker build -t habbits -f apps/habbits/Dockerfile .
```
