---
name: darkmatter-styles
description: Use when building any web UI for Steve - applies his personal dark aesthetic with Commit Mono font, #121113 background, white borders, minimal layout, and max-width centered content. Use for new pages, components, or when asked to match his existing style.
---

# Darkmatter Styles

## Overview

Steve's personal web aesthetic: dark, minimal, monospace. No frameworks, no decorative flourishes. Everything is functional and stark.

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

- **Font:** `"Commit Mono"` (self-hosted .otf), fallback `monospace, sans-serif`
- Applied globally via `* { font-family: ... }`
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

### Font Face Declarations

```css
@font-face {
  font-family: "Commit Mono";
  src: url("/static/fonts/CommitMono-400-Regular.otf") format("opentype");
  font-weight: 400;
  font-style: normal;
}

@font-face {
  font-family: "Commit Mono";
  src: url("/static/fonts/CommitMono-700-Regular.otf") format("opentype");
  font-weight: 700;
  font-style: normal;
}
```

## Base Reset

```css
* {
  padding: 0;
  margin: 0;
  box-sizing: border-box;
  font-family: "Commit Mono", monospace, sans-serif;
  scrollbar-width: none;
  -ms-overflow-style: none;
}

html {
  background: #121113;
  color: #ffffff;
  font-size: 14px;
  line-height: 1.6;
}

html::-webkit-scrollbar {
  display: none;
}
```

## Layout

Single-column, centered, max 700px wide. No top body padding — top spacing comes from header `margin-top`:

```css
body {
  display: flex;
  flex-direction: column;
  justify-content: start;
  align-items: start;
  gap: 1.5rem;
  min-height: 100vh;
  max-width: 700px;
  margin: auto;
  padding: 0 1rem;
}

@media (max-width: 480px) {
  body {
    padding: 1rem;
    gap: 1rem;
  }
}
```

## Header

The header uses a border-bottom separator and `margin-top: 2rem` for top spacing. The site title/logo is **always uppercase**, 28px bold:

```css
.header {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  width: 100%;
  margin-top: 2rem;
  border-bottom: 1px solid #333;
  padding-bottom: 1rem;
}

.logo {
  font-size: 28px;
  font-weight: 700;
  text-decoration: none;
  text-transform: uppercase;
}
```

## Navigation Links

Compact gap, small font:

```css
.links {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  font-size: 12px;
}
```

## Interactive Elements

All inputs, textareas, and buttons match the background — they blend into the surface with only a white border. **No border-radius**, padding uses `0.4rem 0.75rem`:

```css
input, textarea {
  background: #121113;
  color: #ffffff;
  border: 1px solid white;
  padding: 0.4rem 0.75rem;
  font-size: 14px;
  width: 100%;
  border-radius: 0;
}

textarea {
  min-height: 400px;
  resize: vertical;
}

button {
  background: #121113;
  color: #ffffff;
  padding: 0.4rem 0.75rem;
  border: 1px solid white;
  cursor: pointer;
  width: fit-content;
  font-size: 14px;
  border-radius: 0;
}

button:hover, a:hover {
  opacity: 0.7;
}

a {
  color: #ffffff;
  text-decoration: none;
}
```

## Labels

```css
label {
  font-size: 12px;
  opacity: 0.7;
}
```

## Errors

Use a left border accent, not a full box border:

```css
.error {
  color: #ffffff;
  border-left: 2px solid #ffffff;
  padding-left: 0.5rem;
  font-size: 13px;
  opacity: 0.8;
}
```

## List Items

Vertical stacking with bottom borders as dividers, 16px title size:

```css
.item-list {
  display: flex;
  flex-direction: column;
  width: 100%;
}

.item {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  padding: 0.75rem 0;
  border-bottom: 1px solid #333;
}

.item:hover {
  opacity: 0.7;
}

.item-title {
  font-size: 16px;
}

.item-meta {
  font-size: 12px;
  opacity: 0.5;
}
```

## Table Headers

Uppercase, dimmed, lightweight:

```css
th {
  opacity: 0.5;
  font-weight: 400;
  font-size: 12px;
  text-transform: uppercase;
}
```

## Meta Tags

Always include:
```html
<meta name="theme-color" content="#121113" />
```

## What NOT to Do

- No `border-radius` — keep all corners sharp (explicitly set `border-radius: 0` on inputs/buttons)
- No box shadows or drop shadows
- No color other than `#121113`, `#ffffff`, and the gray tones (`#1e1c1f`, `#333`, `#555`)
- **No `color: #888`** — use `opacity` on white text for visual hierarchy instead
- No external font CDNs — fonts are self-hosted
- No utility frameworks (no Tailwind, no Bootstrap)
- No decorative elements, icons, or emojis
