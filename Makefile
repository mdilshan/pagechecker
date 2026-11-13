.PHONY: build run dev test cover

build:
	go build -o bin/server ./cmd/main.go

run:
	go run ./cmd/main.go

dev:
	air

test:
	go test ./cmd/ ./internal/... ./config/ ./views/

# unit tests + the integration test in cmd/
cover:
	go test -coverprofile=cover.out ./cmd/ ./internal/... ./config/ ./views/
	go tool cover -func=cover.out
	@go tool cover -func=cover.out | grep -v "^total" | \
		awk '{n++} $$NF+0 > 0 {c++} END {printf "functions: %d/%d covered (%.1f%%)\n", c, n, c*100/n}'
