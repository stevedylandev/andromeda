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

## Repo discovery

Each entry of `KEPLER_REPO_ROOT` is treated as a candidate:

- `<name>.git/` → bare repo; display name = stripped basename.
- `<name>/.git/` → non-bare repo; display name = dirname.

`description` file in the repo directory shows on the index and repo home
(softserve writes this; the default placeholder is filtered out).
