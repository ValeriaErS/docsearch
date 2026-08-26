package rerank

import (
	"context"
	"docsearch/internal/config"
	"testing"
)

func TestNewReranker(t *testing.T) {
	cfg := &config.Config{}
	cfg.LLM.Provider = "mock"

	reranker := NewReranker(cfg)
	if reranker == nil {
		t.Error("NewReranker() вернул nil")
	}
	if reranker.cfg == nil {
		t.Error("NewReranker() не сохранил конфиг")
	}
}

func TestRerankWithEmptyChunks(t *testing.T) {
	cfg := &config.Config{}
	cfg.LLM.Provider = "mock"
	cfg.Retrieval.TopK = 5

	reranker := NewReranker(cfg)
	ctx := context.Background()

	indices, _, err := reranker.Rerank(ctx, "test", []string{}, 5)

	if err != nil {
		t.Errorf("Rerank() с пустыми чанками ошибка: %v", err)
	}
	if len(indices) != 0 {
		t.Errorf("Rerank() вернул %d индексов, ожидалось 0", len(indices))
	}
}

func TestRerankWithLessChunksThanTopK(t *testing.T) {
	cfg := &config.Config{}
	cfg.LLM.Provider = "mock"
	cfg.Retrieval.TopK = 10

	reranker := NewReranker(cfg)
	ctx := context.Background()

	chunks := []string{"Документ 1", "Документ 2", "Документ 3"}

	indices, _, err := reranker.Rerank(ctx, "test", chunks, 10)

	if err != nil {
		t.Errorf("Rerank() ошибка: %v", err)
	}
	if len(indices) != 3 {
		t.Errorf("Rerank() вернул %d индексов, ожидалось 3", len(indices))
	}
	for i, idx := range indices {
		if idx != i {
			t.Errorf("Индекс [%d] = %d, ожидалось %d", i, idx, i)
		}
	}
}

func TestRerankWithMoreChunksThanTopK(t *testing.T) {
	cfg := &config.Config{}
	cfg.LLM.Provider = "mock"
	cfg.Retrieval.TopK = 5

	reranker := NewReranker(cfg)
	ctx := context.Background()

	chunks := []string{
		"Документ 1 про RAG",
		"Документ 2 про поиск",
		"Документ 3 про эмбеддинги",
		"Документ 4 про векторы",
		"Документ 5 про реранкинг",
		"Документ 6 про гибридный поиск",
		"Документ 7 про BM25",
	}

	indices, _, err := reranker.Rerank(ctx, "Что такое RAG?", chunks, 5)

	if err != nil {
		t.Errorf("Rerank() ошибка: %v", err)
	}
	if len(indices) != 5 {
		t.Errorf("Rerank() вернул %d индексов, ожидалось 5", len(indices))
	}
}

func TestRerankWithNilChunks(t *testing.T) {
	cfg := &config.Config{}
	cfg.LLM.Provider = "mock"
	cfg.Retrieval.TopK = 5

	reranker := NewReranker(cfg)
	ctx := context.Background()

	indices, _, err := reranker.Rerank(ctx, "test", nil, 5)

	if err != nil {
		t.Errorf("Rerank() с nil чанками ошибка: %v", err)
	}
	if len(indices) != 0 {
		t.Errorf("Rerank() вернул %d индексов, ожидалось 0", len(indices))
	}
}

func TestRerankWithTopKZero(t *testing.T) {
	cfg := &config.Config{}
	cfg.LLM.Provider = "mock"
	cfg.Retrieval.TopK = 0

	reranker := NewReranker(cfg)
	ctx := context.Background()

	chunks := []string{"Документ 1", "Документ 2", "Документ 3"}

	indices, _, err := reranker.Rerank(ctx, "test", chunks, 0)

	if err != nil {
		t.Errorf("Rerank() с topK=0 ошибка: %v", err)
	}
	if len(indices) != 3 {
		t.Errorf("Rerank() с topK=0 вернул %d индексов, ожидалось 3", len(indices))
	}
}

func TestRerankWithSingleChunk(t *testing.T) {
	cfg := &config.Config{}
	cfg.LLM.Provider = "mock"
	cfg.Retrieval.TopK = 5

	reranker := NewReranker(cfg)
	ctx := context.Background()

	chunks := []string{"Единственный документ"}

	indices, _, err := reranker.Rerank(ctx, "test", chunks, 5)

	if err != nil {
		t.Errorf("Rerank() с одним чанком ошибка: %v", err)
	}
	if len(indices) != 1 {
		t.Errorf("Rerank() вернул %d индексов, ожидалось 1", len(indices))
	}
	if indices[0] != 0 {
		t.Errorf("Индекс = %d, ожидалось 0", indices[0])
	}
}

func TestRerankWithChunksEqualTopK(t *testing.T) {
	cfg := &config.Config{}
	cfg.LLM.Provider = "mock"
	cfg.Retrieval.TopK = 5

	reranker := NewReranker(cfg)
	ctx := context.Background()

	chunks := []string{
		"Документ 1",
		"Документ 2",
		"Документ 3",
		"Документ 4",
		"Документ 5",
	}

	indices, _, err := reranker.Rerank(ctx, "test", chunks, 5)

	if err != nil {
		t.Errorf("Rerank() ошибка: %v", err)
	}
	if len(indices) != 5 {
		t.Errorf("Rerank() вернул %d индексов, ожидалось 5", len(indices))
	}
}

func TestRerankWithNilConfig(t *testing.T) {
	var cfg *config.Config = nil

	reranker := NewReranker(cfg)
	if reranker == nil {
		t.Error("NewReranker() с nil конфигом вернул nil")
	}
}

func TestMin(t *testing.T) {
	tests := []struct {
		a, b     int
		expected int
	}{
		{5, 10, 5},
		{10, 5, 5},
		{7, 7, 7},
		{0, 5, 0},
		{-5, 3, -5},
		{-10, -5, -10},
	}

	for _, tt := range tests {
		result := min(tt.a, tt.b)
		if result != tt.expected {
			t.Errorf("min(%d, %d) = %d, ожидалось %d", tt.a, tt.b, result, tt.expected)
		}
	}
}

func TestRerankParseResponse(t *testing.T) {
	t.Skip("Skipping parse response test - requires real LLM or mocking")
}