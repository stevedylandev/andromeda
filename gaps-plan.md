# Plan: bring Go ports to parity with Rust originals

## Context

Repo `/home/stevedylandev/Developer/andromeda` has paired Rust + Go versions of 10 apps. Audit (Phase 1) found gaps in 4 Go apps. User wants full parity. Deep-dive (Phase 2) corrected the initial gap list — some "missing" items in jotts-go are already implemented, and the posts EXIF strip referenced earlier does **not** exist in the Rust source either.

This plan covers only the real, confirmed gaps.

---

## Apps at parity (no work)

bookmarks, cellar, easel, feeds, library, og.

---

## Gap 1 — `posts-go`: R2/S3 storage backend

**Rust source to mirror:**
- `apps/posts/src/storage.rs` — `R2Config { bucket, creds, public_url, http }`, methods `from_env`, `put_object`, `delete_object`, `public_url_for`. Uses `rusty_s3 = "0.9"` + reqwest.
- `apps/posts/src/server/mod.rs:552` — init at startup, store `Option<R2Config>` in `AppState`.
- `apps/posts/src/server/handlers/admin.rs:439-522` — `admin_upload_file` routes via `if let Some(r2) = &state.r2`, sets `storage_backend` to `"r2"` or `"local"`, rolls back on failure.
- `apps/posts/src/server/handlers/public.rs:165-199` — `serve_uploaded_file` redirects to `r2.public_url_for(filename)` when backend is r2.

**EXIF strip is NOT in Rust** — drop from scope.

**Go changes (apps/posts-go/):**
- New package `storage/` with interface:
  ```go
  type Backend interface {
      Put(ctx context.Context, key, contentType string, data []byte) error
      Delete(ctx context.Context, key string) error
      PublicURL(key string) string
      Name() string // "local" | "r2"
  }
  ```
- `storage/local.go` — wrap existing FS funcs from `storage.go`.
- `storage/r2.go` — use `github.com/aws/aws-sdk-go-v2/service/s3` with custom endpoint resolver for R2 (`https://<account>.r2.cloudflarestorage.com`). Or `github.com/minio/minio-go/v7` for less ceremony.
- `app.go` — add `Storage storage.Backend` field.
- `main.go` — `if os.Getenv("R2_BUCKET") != "" { storage.NewR2(...) } else { storage.NewLocal(uploadsDir) }`.
- `handlers_admin.go` (~line 301 `adminUploadFile`) — call `a.Storage.Put(...)`, capture `a.Storage.Name()` for DB insert, delete on rollback.
- `handlers_public.go` (~line 156 `serveUploadedFile`) — if `f.StorageBackend == "r2"`, `http.Redirect(... StatusTemporaryRedirect)`.
- `db.go` (~line 395 `createFile`) — pass backend string; column already exists.
- `.env.example` — add `R2_ACCOUNT_ID`, `R2_ACCESS_KEY_ID`, `R2_SECRET_ACCESS_KEY`, `R2_BUCKET`, `R2_PUBLIC_URL`.

---

## Gap 2 — `shrink-go`: EXIF preserve + GPS strip

**Rust source to mirror:** `apps/shrink/src/server.rs:103-207` (`strip_gps_from_exif`). Crates: `img-parts = "0.3"` + `image = "0.25"`. Algorithm:
1. `Jpeg::from_bytes(orig)` → `j.exif().to_vec()` (raw APP1 payload).
2. Re-encode resized JPEG (loses EXIF).
3. Parse raw EXIF: TIFF header (II/MM), read IFD0 offset, walk entries (12 bytes each) for tag `0x8825` (GPS IFD pointer), zero the GPS IFD's entry count (2 bytes at the pointed offset).
4. `out_jpeg.set_exif(Some(exif.into()))`, write final bytes.

**Go changes (apps/shrink-go/):**
- Add deps: `github.com/dsoprea/go-exif/v3` and `github.com/dsoprea/go-jpeg-image-structure/v2`.
- New `exif.go`:
  - `extractExif(orig []byte) []byte` — use go-jpeg-image-structure segment list, return APP1 bytes (skip first 6 `Exif\0\0` prefix).
  - `stripGPS(exif []byte) []byte` — mirror Rust byte-level walk (don't use go-exif builder; cheaper to keep parity).
  - `injectExif(jpeg, exif []byte) []byte` — splice modified APP1 after SOI marker.
- `image.go` / `handlers.go` — wrap compress flow: extract before resize, strip GPS, inject after re-encode.

---

## Gap 3 — `sipp-go`: content wrap toggle (TUI theme is out of scope)

**Already done in Go** (verified via `apps/sipp-go/tui/`): syntect→chroma highlight, clipboard auto-copy on select (`update.go:244-261`), line numbers (`ShowLineNumbers = true`), vim keybindings (`keys.go`).

**Remaining:** Ctrl+W content wrap toggle (Rust `wrap_content: bool`).

**Go changes (apps/sipp-go/tui/):**
- `model.go` — add `wrapContent bool` field.
- `keys.go` — add `WrapToggle` binding (`ctrl+w`).
- `update.go` — when focus is content view/edit and `WrapToggle` matches, flip flag, reset scroll, emit status message.
- `view.go` — when rendering content viewport/textarea, branch on `wrapContent`.

**Custom .tmTheme loading:** Chroma has no native TextMate XML loader. Defer — README already documents the trade-off. Re-open as a follow-up if user wants it; will require either a tmTheme→chroma converter or hand-porting the two themes to chroma `chroma.MustNewStyle`.

---

## Gap 4 — `jotts-go`: complete the TUI editor

**Already done in Go** (verified):
- `cmd_auth.go` — interactive auth + `~/.config/jotts/config.toml` (BurntSushi/toml).
- `cmd_upload.go` — file → note + clipboard.
- `cmd_server.go:29` — startup `sessions.PruneExpired()` (in `crates-go/auth/auth.go`).

**Remaining:** interactive TUI editor. Mirror `apps/jotts/src/tui/{app,events,render}.rs` + `apps/jotts/src/tui.rs`.

**Go changes (apps/jotts-go/tui/):**
- `model.go` — `Focus` enum: List, Content, CreateTitle, CreateContent, EditTitle, EditContent, Search.
- `keys.go` — vim bindings: `hjkl`, `y` (copy content), `Y` (copy share link), `d` (delete with confirm), `c` (new), `e` (edit), `E` (external editor), `o` (open in browser), `/` (search), `?` (help), `q`/`esc` (quit/back).
- `update.go` — mode FSM driving Focus transitions; copy triggers via `atotto/clipboard.WriteAll`; status-line messages.
- `editor.go` (new) — external editor: read `$EDITOR`, write content to `os.CreateTemp`, `exec.Command(editor, path).Run()` with stdio attached to current TTY, re-read file on exit.
- `browser.go` (new) — open `<remote_url>/notes/<short_id>` via `github.com/pkg/browser` (cross-platform).
- `view.go` — two-pane layout (lipgloss `JoinHorizontal`, 30/70), borders/title styles, help line at bottom.
- Markdown render: keep existing `glamour` (acceptable substitute for syntect — agent confirmed parity in behavior).
- `backend.go` — already abstracts local/remote; reuse.

**Deps to add:** `github.com/pkg/browser` (everything else already in go.mod).

---

## Critical files (modified or created)

| App | Path | Action |
|-----|------|--------|
| posts-go | `storage/{interface,local,r2}.go` | new |
| posts-go | `app.go`, `main.go`, `handlers_admin.go`, `handlers_public.go`, `db.go`, `.env.example` | edit |
| shrink-go | `exif.go` | new |
| shrink-go | `image.go`, `handlers.go`, `go.mod` | edit |
| sipp-go | `tui/{model,keys,update,view}.go` | edit |
| jotts-go | `tui/{editor,browser}.go` | new |
| jotts-go | `tui/{model,keys,update,view}.go`, `go.mod` | edit |

---

## Reused existing utilities

- `crates-go/auth/auth.go` — sessions, bearer/session middleware (no changes).
- `crates-go/web` — embedded handler, render helpers (no changes).
- `crates-go/sqlite`, `crates-go/config`, `crates-go/darkmatter` — reused as-is.
- posts-go local FS funcs already in `storage.go` — wrap into `LocalStorage`.
- sipp-go clipboard + chroma flow already wired — only add wrap toggle.
- jotts-go `tui/backend.go` (local + remote impls) — reused.

---

## Execution order (suggested)

1. **shrink-go EXIF** — smallest, self-contained, no schema changes.
2. **sipp-go wrap toggle** — ~20 LoC, fast win.
3. **posts-go R2** — biggest data-path change; needs R2 creds for end-to-end test.
4. **jotts-go TUI** — largest, mostly UI iteration; do last so other parity work isn't blocked.

---

## Verification

Per app:

**shrink-go**
```bash
cd apps/shrink-go && go build ./... && go run .
# upload a JPEG with GPS via the form, download result, run:
exiftool result.jpg | rg -i 'gps|orientation|make|model'
# expect: GPS fields absent, other EXIF present
```

**sipp-go**
```bash
cd apps/sipp-go && go run ./cmd/server
# open snippet, press Ctrl+W, confirm wrap behavior toggles, scroll resets
```

**posts-go**
```bash
cd apps/posts-go && go build ./... && go test ./...
# unset R2 vars: upload via admin → file lands in uploads/, GET /uploads/<f> returns 200
# set R2 vars (test bucket): upload → check bucket contains object, GET returns 307 to R2 public URL
sqlite3 posts.sqlite "select storage_backend from files order by id desc limit 5;"
```

**jotts-go**
```bash
cd apps/jotts-go && go run . server &
go run . tui --remote http://localhost:3000 --api-key $KEY
# walk: create note, edit, y copies content, Y copies link, E opens $EDITOR, o opens browser, d deletes
```

Cross-cutting: `go vet ./...` and `go build ./...` from repo root after each app's changes.

---

## Out of scope (recorded for follow-up)

- sipp-go custom `.tmTheme` loading — needs chroma converter or hand-port; defer per README's existing call-out.
- posts-go EXIF strip on upload — not in Rust either; only do if user asks separately.
