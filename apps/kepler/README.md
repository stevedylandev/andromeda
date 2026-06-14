# kepler

Read-only web view for on-disk git repositories. Drop-in compatible with
softserve's `<data>/repos` layout — point `KEPLER_REPO_ROOT` at it.

## Features

- Repo index with description + last-commit time
- README rendering (goldmark)
- Tree / blob browsing at any ref (branch, tag, SHA)
- Syntax-highlighted source view (chroma)
- Raw file download
- Commit log with pagination
- Single-commit diff view
- Branches + tags page
- Archive download: `.tar.gz`, `.zip`
- Atom feed per repo
- Public read-only JSON API

## API

`GET /api/repos` → JSON list of all repos (no auth, CORS `*`):

```json
{
  "site": "kepler",
  "repos": [
    {
      "name": "andromeda",
      "description": "monorepo",
      "default_ref": "main",
      "last_commit": "2026-06-14T10:00:00Z",
      "url": "https://git.example.com/andromeda",
      "atom_url": "https://git.example.com/andromeda/atom.xml",
      "clone_https": "https://git.example.com/andromeda.git",
      "clone_ssh": "git@git.example.com:andromeda.git"
    }
  ]
}
```

`url` / `atom_url` use the request host (honors `X-Forwarded-Proto`); clone
fields appear only when `KEPLER_CLONE_BASE_URL` / `KEPLER_CLONE_SSH_HOST` are set.

## Run

```bash
KEPLER_REPO_ROOT=~/.local/share/soft-serve/repos go run .
```

Then open <http://127.0.0.1:4747>.

## Env

| Variable | Default | Notes |
|---|---|---|
| `HOST` | `127.0.0.1` | use `0.0.0.0` in Docker |
| `PORT` | `4747` | |
| `KEPLER_REPO_ROOT` | `./repos` | dir of bare repos (`*.git/`) or normal repos |
| `KEPLER_SITE_NAME` | `kepler` | shown in header + feed |
| `KEPLER_BASE_URL` | `http://localhost:4747` | public URL for Open Graph / social meta tags |
| `KEPLER_CLONE_BASE_URL` | _(empty)_ | HTTPS entry in the repo-home Clone menu; URL = `<base>/<repo>.git`. Point at the host that actually serves git (e.g. softserve), not kepler |
| `KEPLER_CLONE_SSH_HOST` | _(empty)_ | SSH (scp-style) entry in the Clone menu; URL = `<user@host>:<repo>.git`, e.g. `git@git.example.com`. Include the user |

The repo-home **Clone** dropdown shows whichever of the two are set; if both
are empty the button is hidden.

## Repo discovery

Each entry of `KEPLER_REPO_ROOT` is treated as a candidate:

- `<name>.git/` → bare repo; display name = stripped basename.
- `<name>/.git/` → non-bare repo; display name = dirname.

`description` file in the repo directory shows on the index and repo home
(softserve writes this; the default placeholder is filtered out).
