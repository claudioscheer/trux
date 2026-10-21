# Trux documentation site

The source of <https://claudioscheer.github.io/trux/>, built with [Hugo](https://gohugo.io) (extended, 0.140 or newer)
and the [Lotus Docs](https://github.com/colinwilson/lotusdocs) theme, which Hugo fetches as a Go module (so Go is needed
too).

```bash
cd website
hugo server          # http://localhost:1313/trux/
hugo --gc --minify   # build into website/public/
```

| Path | Contents |
| --- | --- |
| `hugo.toml` | site configuration, theme module, and mounts of `../docs`, `../examples`, and `outputs/` |
| `content/docs/` | the documentation pages; sidebar order comes from each page's `weight` |
| `content/docs/design-notes/_content.gotmpl` | publishes the design notes in `../docs/*.md` without moving them |
| `data/landing.yaml` | the landing page |
| `layouts/` | the `include` shortcode and small overrides of Lotus Docs templates (each file says what changed) |
| `static/docs/js/components/prism-trux.min.js` | Trux syntax highlighting for Prism |
| `outputs/` | real program output shown on the site, written by `update-example-outputs.sh` (CI runs it with `--check`) |

More detail: [Working on the documentation site](https://claudioscheer.github.io/trux/docs/internals/contributing/#working-on-the-documentation-site).
