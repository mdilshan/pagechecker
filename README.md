# pagechecker

analyze a web page. shows html version, title, headings, links and whether it has a login form.

## run dev server

install tools first

```sh
go install github.com/air-verse/air@latest
go install github.com/a-h/templ/cmd/templ@latest
```

then run it, it reloads on code changes

```sh
make dev
```

open http://localhost:8000

## run prod server

pprof is on in the default config, so turn it off for prod. make a `prod.ini`
with only the values you want to change

```ini
[server]
pprof = false
```

then mount it and point `PAGECHECKER_CONFIG` to it

```sh
docker build -t pagechecker .
docker run --rm -p 8000:8000 -e GIN_MODE=release \
  -v $(pwd)/prod.ini:/etc/pagechecker.ini:ro \
  -e PAGECHECKER_CONFIG=/etc/pagechecker.ini \
  pagechecker
```

open http://localhost:8000

## config

defaults are in `config/config.ini` and built into the binary. `PAGECHECKER_CONFIG` env
points to another ini file, its values override the defaults. works without
docker too

```sh
PAGECHECKER_CONFIG=./prod.ini go run ./cmd
```

## pprof

when `pprof = true` it's served on the same port

open http://localhost:8000/debug/pprof/ to see the counts

goroutines, with stacks

```
http://localhost:8000/debug/pprof/goroutine?debug=1
```

memory still in use (`gc=1` runs gc first so freed memory is not counted)

```sh
go tool pprof -sample_index=inuse_space "http://localhost:8000/debug/pprof/heap?gc=1"
```

cpu for 30 seconds, submit some pages while it runs

```sh
go tool pprof "http://localhost:8000/debug/pprof/profile?seconds=30"
```

type `top` in the pprof prompt to see the biggest ones. the heap count on the
index page is not memory, it's the number of places that allocated.
