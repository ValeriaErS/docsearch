.PHONY: test build demo

test:
	go test ./...

build:
	go build -o bin/docsearch.exe ./cmd/docsearch

demo:
	go run ./cmd/docsearch index
	go run ./cmd/docsearch ask "Как установить Linux?"
