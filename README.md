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

```sh
docker build -t pagechecker .
docker run --rm -p 8000:8000 -e GIN_MODE=release pagechecker
```

open http://localhost:8000
