.PHONY: test build demo eval clean

test:
	go test ./...

build:
	go build -o bin/docsearch.exe ./cmd/docsearch

demo:
	@echo "Демо режим (использует mock конфиг)"
	mkdir docs\demo 2>nul || echo Папка уже существует
	copy testdata\control\*.md docs\demo\ 2>nul || echo Файлы уже есть
	go run ./cmd/docsearch index --user demo
	go run ./cmd/docsearch ask "What is RAG?" --user demo --out demo_result.json
	go run ./cmd/docsearch ask "How to install DocSearch?" --user demo
	go run ./cmd/docsearch ask "How to install Linux?" --user demo
	@echo "Готово. Результат в demo_result.json"

eval:
	go run ./cmd/docsearch eval --user demo

clean:
	del /f /q bin\docsearch.exe 2>nul
	del /f /q .docsearch_index_*.json 2>nul
	del /f /q eval_results.json 2>nul
	del /f /q demo_result.json 2>nul
	rmdir /s /q docs\demo 2>nul