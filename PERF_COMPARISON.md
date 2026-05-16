# Andromeda: Rust vs Go Performance Comparison

Comparison of the Rust apps (current production, running in Docker) against the in-progress Go rewrites on the `chore/go-rewrite` branch.

Date: 2026-05-16
Host: Linux 7.0.3-arch1-2

## Methodology

- **Binary size**: `cargo build --release --workspace` (with `[profile.release]` defaults) vs each `go build` artifact in `apps/*-go/`. Raw stripped/unstripped binaries on disk, no UPX or other post-processing.
- **Lines of code**: Total lines in `apps/<app>/src/**/*.rs` vs `apps/<app>-go/**/*.go`. Includes blanks and comments. Shared crates counted separately.
- **Dependencies**: Direct deps parsed from `Cargo.toml` `[dependencies]` vs `require ( ... )` blocks in `go.mod` (separating direct vs total-with-indirect).
- **RAM**: Two passes.
  1. *Production snapshot* — `docker stats` against the long-running Rust containers (not apples-to-apples; reflects accumulated state).
  2. *Fair cold start* — local release binaries (Rust from `target/release/`, Go from `apps/*-go/`) launched on unused ports with `PORT`/`HOST` overrides, sampled `VmRSS` from `/proc/<pid>/status` after a 2s warmup, then terminated. Same conditions, no traffic, fresh sqlite.

## Binary Size

| App       | Rust   | Go     | Go vs Rust |
|-----------|--------|--------|------------|
| bookmarks | 14M    | 20M    | +43%       |
| cellar    | 16M    | 20M    | +25%       |
| easel     | 15M    | 20M    | +33%       |
| feeds     | 16M    | 23M    | +44%       |
| jotts     | 19M    | 21M    | +11%       |
| library   | 13M    | 20M    | +54%       |
| og        | 12M    | 14M    | +17%       |
| posts     | 16M    | 21M    | +31%       |
| shrink    | 8.2M   | 14M    | +71%       |
| sipp      | 16M    | 22M    | +38%       |

**Winner: Rust** — smaller in every app, average ~35% smaller.

## Lines of Code

Raw totals:

| App       | Rust  | Go    | Go ratio |
|-----------|-------|-------|----------|
| bookmarks | 850   | 756   | 0.89x    |
| cellar    | 2010  | 1241  | 0.62x    |
| easel     | 1156  | 946   | 0.82x    |
| feeds     | 2193  | 1981  | 0.90x    |
| jotts     | 2248  | 526   | 0.23x    |
| library   | 1021  | 882   | 0.86x    |
| og        | 399   | 295   | 0.74x    |
| posts     | 3010  | 2140  | 0.71x    |
| shrink    | 389   | 166   | 0.43x    |
| sipp      | 2455  | 655   | 0.27x    |

### TUI/CLI adjustment

`jotts` and `sipp` Rust binaries include a full TUI/CLI alongside the server. Go ports are server-only. Stripping TUI sources for a fair compare:

| App   | Rust server only | Rust TUI/CLI | Go    | Go ratio (server) |
|-------|------------------|--------------|-------|--------------------|
| jotts | 1063             | 1185         | 526   | 0.49x              |
| sipp  | 1203             | 1252         | 655   | 0.54x              |

All other apps are server-only on both sides — raw numbers are already fair.

### Shared modules

| Side  | Modules                                  | Total LOC |
|-------|------------------------------------------|-----------|
| Rust  | `crates/auth` (371), `crates/db` (833), `crates/darkmatter-css` (54) | 1258 |
| Go    | `crates-go/auth` (207), `crates-go/config` (58), `crates-go/darkmatter` (54), `crates-go/web` (72) | 391 |

**Winner: Go** — consistently fewer lines (~30–50% less for server code).

## Dependencies

| App       | Rust direct | Go direct | Go +indirect |
|-----------|-------------|-----------|---------------|
| bookmarks | 22          | 6         | 17            |
| cellar    | 24          | 5         | 16            |
| easel     | 19          | 4         | 14            |
| feeds     | 25          | 7         | 25            |
| jotts     | 29          | 6         | 17            |
| library   | 20          | 5         | 16            |
| og        | 15          | 4         | 4             |
| posts     | 26          | 6         | 17            |
| shrink    | 11          | 4         | 4             |
| sipp      | 32          | 6         | 18            |

**Winner: Go** — far fewer direct deps; stdlib carries weight. Rust transitive trees would be much larger again if expanded via `cargo tree`.

## RAM Usage

### Production snapshot (Rust in Docker, idle)

Long-running containers from `docker stats --no-stream`. Includes accumulated state (background pollers in `cellar`/`feeds` inflate RSS).

| App       | RSS    |
|-----------|--------|
| og        | 7.5M   |
| shrink    | 6.3M   |
| jotts     | 13.0M  |
| library   | 13.1M  |
| sipp      | 13.3M  |
| bookmarks | 22.1M  |
| posts     | 23.2M  |
| easel     | 31.5M  |
| feeds     | 52.1M  |
| cellar    | 64.6M  |

Not comparable to a cold Go binary — different uptime and workload.

### Fair cold start (both sides, alt ports, 2s warmup)

| App       | Rust    | Go      | Winner          |
|-----------|---------|---------|-----------------|
| bookmarks | 11.2M   | 13.6M   | Rust −18%       |
| cellar    | 10.5M   | 12.9M   | Rust −19%       |
| easel     | 21.1M   | 12.8M   | **Go −39%**     |
| feeds     | 11.6M   | 14.3M   | Rust −19%       |
| jotts     | 8.6M    | 14.2M   | Rust −39%       |
| library   | 10.1M   | 12.6M   | Rust −20%       |
| og        | 8.1M    | 10.1M   | Rust −20%       |
| posts     | 10.8M   | 14.9M   | Rust −28%       |
| shrink    | 5.6M    | 9.9M    | Rust −44%       |
| sipp      | 10.1M   | 19.6M   | Rust −48%       |

**Winner: Rust** — 9 of 10 apps. Average ~28% less RAM cold idle. Easel anomaly: Rust eagerly loads classifications/exclusion lists at boot.

## Summary

| Metric                       | Winner | Magnitude              |
|------------------------------|--------|------------------------|
| Binary size                  | Rust   | ~35% smaller avg       |
| Lines of code (server only)  | Go     | ~30–50% fewer          |
| Direct dependencies          | Go     | 4–7 vs 11–32           |
| RAM (cold start, single user)| Rust   | ~28% less avg, 9/10    |

### Context: single-user deployment

Apps are single-user. Cold-start RAM is the right benchmark — no concurrent load spike, no GC pressure differences under traffic, idle is the dominant state.

- Rust: smaller binaries, lower idle RAM, more code, more deps.
- Go: less code, fewer deps, larger binaries, higher idle RAM.

Tradeoff is roughly: pay in source-code volume + dep count to get smaller binaries and lower memory; or pay in binary size + RAM to get less code to maintain.
