.PHONY: test build demo eval clean install

ifeq ($(OS),Windows_NT)
    DETECTED_OS := windows
else
    UNAME_S := $(shell uname -s 2>/dev/null || echo Unknown)
    ifeq ($(UNAME_S),Linux)
        DETECTED_OS := linux
    else ifeq ($(UNAME_S),Darwin)
        DETECTED_OS := darwin
    else
        DETECTED_OS := unknown
    endif
endif

ifeq ($(DETECTED_OS),windows)
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

clean:
	-$(RM) $(BINARY)
	-$(RM) .docsearch_index_*.json
	-$(RM) eval_results.json
	-$(RM) demo_result.json
	-$(RMDIR) tmp

install: build
	$(COPY) $(BINARY) docsearch$(EXE)