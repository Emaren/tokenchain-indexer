.PHONY: run build test

run:
	go run ./cmd/tokenchain-indexer

build:
	go build ./...

test:
	go test ./...
