.PHONY: build run dev

build:
	go build -o bin/server ./cmd/main.go

run:
	go run ./cmd/main.go

dev:
	air
