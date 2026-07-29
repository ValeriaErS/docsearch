.PHONY: test build demo eval clean install compare

# Определяем ОС
ifeq ($(OS),Windows_NT)
    EXE   := .exe
    RM    := del /Q /F
    RMDIR := rmdir /S /Q
    COPY  := copy
    MKDIR := mkdir
else
    EXE   :=
    RM    := rm -f
    RMDIR := rm -rf
    COPY  := cp
    MKDIR := mkdir -p
endif

BINARY      := bin/docsearch$(EXE)
MOCK_CONFIG := configs/config.mock.yml

test:
	go test ./...

build:
	-$(MKDIR) bin
	go build -o $(BINARY) ./cmd/docsearch

demo:
	go run ./cmd/docsearch demo --config $(MOCK_CONFIG)

eval:
	go run ./cmd/docsearch eval --user demo --dataset testdata/control/questions.jsonl --config $(MOCK_CONFIG)

compare:
	go run ./cmd/docsearch compare --user demo --config $(MOCK_CONFIG)

clean:
	-$(RM) $(BINARY)
	-$(RM) .docsearch_index_*.json
	-$(RM) eval_results.json
	-$(RM) demo_result.json
	-$(RMDIR) tmp

install: build
	$(COPY) $(BINARY) docsearch$(EXE)