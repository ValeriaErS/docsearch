.PHONY: test build demo eval clean

test:
	go test ./...

build:
	go build -o bin/docsearch.exe ./cmd/docsearch

demo:
	go run ./cmd/docsearch index
	go run ./cmd/docsearch ask "Как установить Linux?"
	go run ./cmd/docsearch index --user Тест
	go run ./cmd/docsearch ask "Что такое FileAuditor?" --user Тест

eval:
	go run ./cmd/docsearch eval

clean:
	rm -f docsearch.exe .docsearch_index.json eval_results.json