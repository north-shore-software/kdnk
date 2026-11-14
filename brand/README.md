# kdnk brand assets

All letters are converted to vector outlines (JetBrains Mono ExtraBold / Regular, SIL Open Font License), so no font is needed.

| File | Use |
|---|---|
| `kdnk-wordmark-light.svg` / `-dark.svg` (+ `@2x.png`) | The wordmark, transparent background |
| `kdnk-banner-light.svg` / `-dark.svg` (+ `@2x.png`) | README header, 1280×320 |
| `kdnk-icon-light.svg` / `-dark.svg` (+ `-512.png`) | GitHub org/repo avatar, docs logo tile |
| `kdnk-social-preview-light.png` / `-dark.png` | GitHub › Settings › Social preview (1280×640) |
| `favicon.svg` | Docs site favicon; switches to dark automatically |
| `favicon.ico`, `favicon-32.png`, `favicon-16.png`, `apple-touch-icon.png` | Fallback icons |

## Colours

| | Light | Dark |
|---|---|---|
| Background | `#F4F3EE` | `#16181D` |
| Frame and letters | `#16181D` | `#F4F3EE` |
| Box fill | `#FFFFFF` | `#23262D` |
| Apostrophe and shadow | `#C9531F` | `#F07B45` |

## README header (follows GitHub's light/dark theme)

```html
<picture>
  <source media="(prefers-color-scheme: dark)" srcset="docs/assets/kdnk-banner-dark.svg">
  <img alt="kdnk" src="docs/assets/kdnk-banner-light.svg">
</picture>
```

## Docs site `<head>`

```html
<link rel="icon" href="/favicon.svg" type="image/svg+xml">
<link rel="icon" href="/favicon.ico" sizes="32x32">
<link rel="apple-touch-icon" href="/apple-touch-icon.png">
```
