# build stage
FROM golang:1.26 AS build

WORKDIR /src

# *_templ.go files are not committed
RUN go install github.com/a-h/templ/cmd/templ@v0.3.1118

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN templ generate
RUN CGO_ENABLED=0 go build -o /out/server ./cmd

# run stage, static binary only (css and config are embedded)
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=build /out/server /server

EXPOSE 8000
ENTRYPOINT ["/server"]

  # docker build -t pagechecker .
  # docker run --rm -p 8000:8000 -e GIN_MODE=release pagechecker

