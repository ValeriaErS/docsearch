.PHONY: test build demo eval clean

test:
	go test ./...

build:
	go build -o bin/docsearch.exe ./cmd/docsearch

demo:
	@echo "Создаем папку для пользователя demo"
	mkdir docs\demo 2>nul || echo Папка уже существует
	@echo "Копируем тестовые документы"
	copy testdata\control\*.md docs\demo\ 2>nul || echo Файлы уже есть
	@echo "Индексируем документы"
	go run ./cmd/docsearch index --user demo
	@echo "Задаем вопрос 1"
	go run ./cmd/docsearch ask "What is RAG?" --user demo --out demo_result.json
	@echo "Задаем вопрос 2"
	go run ./cmd/docsearch ask "How to install DocSearch?" --user demo
	@echo "Задаем вопрос 3 (должен сказать что нет ответа)"
	go run ./cmd/docsearch ask "How to install Linux?" --user demo

eval:
	go run ./cmd/docsearch eval --user demo

clean:
	del /f /q bin\docsearch.exe 2>nul
	del /f /q .docsearch_index_*.json 2>nul
	del /f /q eval_results.json 2>nul
	del /f /q demo_result.json 2>nul
	rmdir /s /q docs\demo 2>nul