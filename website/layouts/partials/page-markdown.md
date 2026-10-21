{{- /*
  The Markdown version of a docs page, used by the "markdown" output format
  (…/index.md), the copy button, and llms-full.txt.

  It starts from the page source and rewrites this site's shortcodes into plain
  Markdown, so agents get real code instead of shortcode calls. When you add a
  shortcode to the content, add a pattern for it here; the build fails if
  shortcode syntax is left in the output (see the check below).
*/ -}}
{{- $md := .RawContent -}}

{{- /* {{< include "path[:from:to]" "lang" >}}: the file's contents as fenced code. */ -}}
{{- range findRESubmatch `\{\{<\s*include\s+"([^"]+)"(?:\s+"([^"]+)")?\s*>\}\}` $md -}}
  {{- $spec := index . 1 -}}
  {{- $parts := split $spec ":" -}}
  {{- $path := index $parts 0 -}}
  {{- $asset := cond (hasPrefix $path "website/outputs/") (replace $path "website/outputs/" "outputs/" 1) $path -}}
  {{- $res := resources.Get $asset -}}
  {{- if not $res }}{{ errorf "page-markdown: include %q not found" $path }}{{ end -}}
  {{- $text := $res.Content -}}
  {{- if ge (len $parts) 3 -}}
    {{- $lines := split $text "\n" -}}
    {{- $from := sub (int (index $parts 1)) 1 -}}
    {{- $text = delimit (first (sub (int (index $parts 2)) $from) (after $from $lines)) "\n" -}}
  {{- end -}}
  {{- $lang := index . 2 | default "text" -}}
  {{- $md = replace $md (index . 0) (printf "```%s\n%s\n```" $lang (strings.TrimRight "\n" $text)) 1 -}}
{{- end -}}

{{- /* Alerts become plain paragraphs (their bold title line stays). */ -}}
{{- $md = replaceRE `\{\{%\s*alert[^%]*%\}\}` "" $md -}}
{{- $md = replaceRE `\{\{%\s*/alert\s*%\}\}` "" $md -}}
{{- /* Collapsible blocks and tabs become a bold label followed by their content. */ -}}
{{- $md = replaceRE `\{\{<\s*details\s+summary="([^"]*)"\s*>\}\}` "**$1**" $md -}}
{{- $md = replaceRE `\{\{<\s*/details\s*>\}\}` "" $md -}}
{{- $md = replaceRE `\{\{<\s*/?tabs[^>]*>\}\}` "" $md -}}
{{- $md = replaceRE `\{\{%\s*tab\s+tabName="([^"]*)"\s*%\}\}` "**$1**" $md -}}
{{- $md = replaceRE `\{\{%\s*/tab\s*%\}\}` "" $md -}}
{{- /* Escaped shortcode examples (written with the comment-style escape) show the shortcode literally. */ -}}
{{- $md = replaceRE `\{\{<\s*/\*\s*(.*?)\s*\*/\s*>\}\}` "{{< $1 >}}" $md -}}
{{- /* Explicit heading IDs ("## Title {#id}") are site markup, not content. */ -}}
{{- $md = replaceRE `(?m)^(#{1,6} .*?)\s*\{#[\w-]+\}[ \t]*$` "$1" $md -}}
{{- /* Site-relative links need the full URL outside the site. */ -}}
{{- $md = replaceRE `\]\(/docs/` (printf "](%sdocs/" site.BaseURL) $md -}}
{{- $md = replaceRE `\n{3,}` "\n\n" $md | strings.TrimSpace -}}

{{- /* Fail the build on shortcode syntax that no pattern above handled. */ -}}
{{- $leftover := replaceRE "`[^`\n]*`" "" $md -}}
{{- with findRE `\{\{[<%]` $leftover 1 -}}
  {{- errorf "page-markdown: unconverted shortcode syntax in %s; add a pattern to layouts/partials/page-markdown.md" $.RelPermalink -}}
{{- end -}}

# {{ .Title }}
{{ with .Description }}
> {{ . }}
{{ end }}
Source: {{ .Permalink }}

{{ $md }}
