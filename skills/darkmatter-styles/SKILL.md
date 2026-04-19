---
name: darkmatter-styles
description: Applies the Darkmatter aesthetic - Commit Mono font, #121113 background, white borders, minimal layout, and max-width centered content. Use for new pages, components, or when asked to match the Andromeda style.
---

# Darkmatter Styles

## Overview

The Darkmatter aesthetic: dark, minimal, monospace. No frameworks, no decorative flourishes. Everything is functional and stark.

The canonical reset, tokens, fonts, and component styles ship in the [`andromeda-darkmatter-css`](https://github.com/stevedylandev/andromeda/tree/main/crates/darkmatter-css) crate. **Do not duplicate the CSS per app** — mount the crate's router and link its stylesheet.

## Using the Crate

Add to `Cargo.toml`:

```toml
[dependencies]
andromeda-darkmatter-css = { path = "../../crates/darkmatter-css" }
```

Mount in the Axum app:

```rust
let app = Router::new()
    .route("/", get(index))
    .merge(andromeda_darkmatter_css::router());
```

Link from templates:

```html
<link rel="stylesheet" href="/assets/darkmatter.css" />
<meta name="theme-color" content="#121113" />
```

The crate serves:

- `/assets/darkmatter.css` — full stylesheet (reset + tokens + components)
- `/assets/fonts/CommitMono-400-Regular.otf`, `…-700-Regular.otf`
- `/darkmatter` — live component gallery (reference when building)

## Core Palette

| Token | Value | Usage |
|-------|-------|-------|
| Background | `#121113` | All surfaces — html, inputs, buttons, textarea |
| Foreground | `#ffffff` | All text and borders |
| Border | `1px solid white` | Inputs, buttons, textarea |
| Gray Dark | `#1e1c1f` | Code block backgrounds |
| Gray Mid | `#333` | Dividers, list item borders, section separators |
| Gray Light | `#555` | Tertiary borders (blockquote borders) |

**No accent colors, no gradients.** Background, white, and grays only.

### Visual Hierarchy via Opacity (NOT gray color values)

Use opacity on white text instead of gray hex colors for secondary/tertiary text:

| Level | Opacity | Usage |
|-------|---------|-------|
| Primary | 1.0 | Headings, body text, links |
| Secondary | 0.7 | Labels, form labels, blockquotes |
| Tertiary | 0.5 | Nav links dimmed, table headers, dates, metadata, empty states |
| Muted | 0.3 | Null/placeholder values |
| Error | 0.8 | Error messages |

**Do NOT use `color: #888` for secondary text.** Always use `opacity` on white text instead.

## Typography

- **Font:** `"Commit Mono"` (served by the crate), fallback `monospace, sans-serif`
- Applied globally via the crate's `* { font-family: ... }`
- **Body font-size:** 14px
- **Line-height:** 1.6

### Font Size Scale

| Size | Usage |
|------|-------|
| 28px | Site logo/title (bold, uppercase) |
| 18px | Markdown h1 |
| 16px | Markdown h2, note/item titles, primary labels |
| 15px | Markdown h3 |
| 14px | Body text, inputs, buttons, markdown h4-h6 |
| 13px | Inline code, error messages |
| 12px | Nav links, form labels, metadata, dates, table headers, action links |

## Shared Component Classes

The stylesheet ships ready-to-use classes. Prefer these over writing new rules:

- Layout: `.header`, `.logo`, `.links`, `.footer`
- Forms: `.form`, `.form-row`, `.form-field`, `.form-actions`, `.checkbox-field`, `.switch`, `.switch-slider`
- Buttons: `button`, `.btn`, `.link-button`, `.link-button.danger`
- Lists: `.item-list` / `.item` / `.item-title` / `.item-meta`
- Admin lists: `.admin-list`, `.admin-list-item`, `.admin-toolbar`, `.admin-list-actions`
- Feedback: `.error`, `.success`, `.empty`
- Tags: `.tag`, `.status-badge`, `.status-published`, `.status-draft`
- Misc: `.spinner`, `.scroll-x`, `.hidden`, `.inline-form`

Visit `/darkmatter` on any app that mounts the crate to see live examples.

## Writing New Styles

Only add app-specific CSS when the shared sheet does not cover the pattern. When you do:

- Match the palette and opacity scale above
- `border-radius: 0` everywhere
- Keep selectors shallow and class-based
- Prefer extending existing classes over inventing new ones

## What NOT to Do

- Do not copy Darkmatter CSS into an app's static assets — link the crate's `/assets/darkmatter.css` instead
- No `border-radius` — keep all corners sharp
- No box shadows or drop shadows
- No color other than `#121113`, `#ffffff`, and the gray tones (`#1e1c1f`, `#333`, `#555`)
- **No `color: #888`** — use `opacity` on white text for visual hierarchy instead
- No external font CDNs — fonts come from the crate
- No utility frameworks (no Tailwind, no Bootstrap)
- No decorative elements, icons, or emojis
