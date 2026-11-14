# notes

## build and deploy steps

1. install go 1.26, plus `templ` and `air` for dev
2. `templ generate` turns the `.templ` views into go code (`*_templ.go` is not
   committed)
3. `make dev` runs the app with live reload, `make test` runs the tests,
   `make cover` shows coverage
4. `docker build -t pagechecker .` builds the image. it's a multi stage build,
   the final image only has the static binary (css and config are embedded)
5. `docker run` with a mounted `prod.ini` and `PAGECHECKER_CONFIG` to override
   the defaults, see the README

## assumptions and decisions

- **ssrf is left for a future release.** we assume this is an internal tool, so
  users are trusted and the server is allowed to call any url, internal ones
  included
- **login detection expects a standard email/password login page.** a page has
  a login form if it has an `<input type="password">`. the check sits in one
  place in the analyzer so it's easy to extend
- **built for bulk analysis later.** we assume the tool may be used to analyze
  many pages at once in the future, so we laid the groundwork for low memory
  use: the page body is streamed into the tokenizer and never kept in memory,
  and link checks are bounded per request and across the whole server
- **no support for js rendered sites (single page apps).** we only read the
  html the server sends. pages that build their content with js, or show a bot
  challenge, will look empty or incomplete
- **inaccessible** means the link answered 400 or above, or could not be
  reached. redirects are followed, so a 301 to a working page is fine.
  `mailto:`, `tel:`, `javascript:` links are not checked and count as
  inaccessible, malformed links too
- **link counting** by deafult it counts only <a> and <img> links only.
```go
var linkAttrs = map[string]string{
	"a":   "href",
	"img": "src",
}
```


## possible improvements

- ssrf protection, block private and internal ip ranges
- render js sites with a headless browser, so single page apps and bot
  challenges work
- better login detection: url paths like `/login`, social login buttons,
  pages that redirect to a login (401/403)
- skip duplicate links before checking them
- cache link results for a while, so the same link isn't checked again for
  every page
- respect `<base href>` when resolving relative links
- limit requests per target host, so one site isn't hammered
- bulk analysis api to submit many urls at once
- prometheus metrics (pages analyzed, links checked, durations)
- request logs through `slog` too, gin still uses its own logger
- pprof off by default, on only in dev
