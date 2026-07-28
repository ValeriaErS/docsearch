.PHONY: test build demo eval clean

test:
	go test ./...

build:
	go build -o bin/docsearch.exe ./cmd/docsearch

demo:
	@echo "Демо режим (mock)"
	del .docsearch_index_demo.json 2>nul || echo Индекс не найден
	mkdir docs\demo 2>nul || echo Папка уже существует
	copy testdata\control\*.md docs\demo\ 2>nul || echo Файлы уже есть
	go run ./cmd/docsearch demo --config configs/config.mock.yml
	@echo "Готово. Результат в demo_result.json"

eval:
	go run ./cmd/docsearch eval --user demo

clean:
	del /f /q bin\docsearch.exe 2>nul
	del /f /q .docsearch_index_*.json 2>nul
	del /f /q eval_results.json 2>nul
	del /f /q demo_result.json 2>nul
	rmdir /s /q docs\demo 2>nul

install: build
	@echo "Устанавливаю docsearch"
	copy bin\docsearch.exe docsearch.exe
	@echo "Готово! Теперь можно запускать: .\docsearch.exe"