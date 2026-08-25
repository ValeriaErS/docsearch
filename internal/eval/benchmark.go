package eval

import (
	"bufio"
	"context"
	"docsearch/internal/config"
	"docsearch/internal/rag"
	"docsearch/internal/vector"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type BenchQuestion struct {
	Query        string   `json:"query"`
	ExpectedDocs []string `json:"expected_docs"`
}

type BenchResult struct {
	Query      string   `json:"query"`
	FoundDocs  []string `json:"found_docs"`
	RecallAt5  float64  `json:"recall_at_5"`
	RecallAt10 float64  `json:"recall_at_10"`
	MRR        float64  `json:"mrr"`
	LatencyMs  int64    `json:"latency_ms"`
	TokensUsed int      `json:"tokens_used"`
}

type BenchSummary struct {
	TotalQueries  int     `json:"total_queries"`
	AvgRecallAt5  float64 `json:"avg_recall_at_5"`
	AvgRecallAt10 float64 `json:"avg_recall_at_10"`
	AvgMRR        float64 `json:"avg_mrr"`
	AvgLatencyMs  int64   `json:"avg_latency_ms"`
	AvgTokensUsed int     `json:"avg_tokens_used"`
	SuccessRate   float64 `json:"success_rate"`
}

func RunBenchmark(cfg *config.Config, userID string, vectorClient vector.VectorStore) error {
	questions, err := loadQuestions("testdata/control/questions.jsonl")
	if err != nil {
		return fmt.Errorf("ошибка загрузки вопросов: %w", err)
	}

	fmt.Printf("\nЗАПУСК БЕНЧМАРКА\n")
	fmt.Printf("Вопросов: %d\n", len(questions))
	fmt.Println(strings.Repeat("-", 60))

	var results []BenchResult
	var totalRecall5, totalRecall10, totalMRR float64
	var totalLatency int64
	var totalTokens int
	var successCount int

	for i, q := range questions {
		fmt.Printf("[%d/%d] %s\n", i+1, len(questions), q.Query)

		start := time.Now()
		_, docs, _, _, _, _, tokensUsed, _ := rag.Ask(
			context.Background(),
			*cfg,
			q.Query,
			userID,
			[]map[string]string{},
			vectorClient,
		)

		fmt.Printf("  Найденные документы: %v\n", docs)
		fmt.Printf("  Ожидаемые документы: %v\n", q.ExpectedDocs)

		latency := time.Since(start).Milliseconds()

		recall5 := calcRecall(docs, q.ExpectedDocs, 5)
		recall10 := calcRecall(docs, q.ExpectedDocs, 10)
		mrr := calcMRR(docs, q.ExpectedDocs)

		success := false
		expectedSet := make(map[string]bool)
		for _, exp := range q.ExpectedDocs {
			expectedSet[exp] = true
		}
		for _, found := range docs {
			if expectedSet[found] {
				success = true
				break
			}
		}
		if success {
			successCount++
		}

		results = append(results, BenchResult{
			Query:      q.Query,
			FoundDocs:  docs,
			RecallAt5:  recall5,
			RecallAt10: recall10,
			MRR:        mrr,
			LatencyMs:  latency,
			TokensUsed: tokensUsed,
		})

		totalRecall5 += recall5
		totalRecall10 += recall10
		totalMRR += mrr
		totalLatency += latency
		totalTokens += tokensUsed
	}

	summary := BenchSummary{
		TotalQueries:  len(questions),
		AvgRecallAt5:  totalRecall5 / float64(len(questions)),
		AvgRecallAt10: totalRecall10 / float64(len(questions)),
		AvgMRR:        totalMRR / float64(len(questions)),
		AvgLatencyMs:  totalLatency / int64(len(questions)),
		AvgTokensUsed: totalTokens / len(questions),
		SuccessRate:   float64(successCount) / float64(len(questions)) * 100,
	}

	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("РЕЗУЛЬТАТЫ БЕНЧМАРКА")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("Всего вопросов:     %d\n", summary.TotalQueries)
	fmt.Printf("Успешных ответов:   %d (%.1f%%)\n", successCount, summary.SuccessRate)
	fmt.Printf("Recall@5:          %.2f%%\n", summary.AvgRecallAt5*100)
	fmt.Printf("Recall@10:         %.2f%%\n", summary.AvgRecallAt10*100)
	fmt.Printf("MRR:               %.3f\n", summary.AvgMRR)
	fmt.Printf("Средняя задержка:  %d мс\n", summary.AvgLatencyMs)
	fmt.Printf("Среднее токенов:   %d\n", summary.AvgTokensUsed)
	fmt.Println(strings.Repeat("=", 60))

	jsonData, _ := json.MarshalIndent(results, "", "  ")
	os.WriteFile("benchmark_results.json", jsonData, 0644)
	summaryJSON, _ := json.MarshalIndent(summary, "", "  ")
	os.WriteFile("benchmark_summary.json", summaryJSON, 0644)
	fmt.Println("\nРезультаты сохранены в benchmark_results.json и benchmark_summary.json")

	return nil
}

func loadQuestions(path string) ([]BenchQuestion, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var questions []BenchQuestion
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var q BenchQuestion
		if err := json.Unmarshal([]byte(line), &q); err != nil {
			continue
		}
		questions = append(questions, q)
	}
	return questions, scanner.Err()
}

func calcRecall(foundDocs, expectedDocs []string, k int) float64 {
	if len(expectedDocs) == 0 {
		return 0
	}

	if len(foundDocs) == 0 {
		return 0
	}

	limit := k
	if len(foundDocs) < limit {
		limit = len(foundDocs)
	}

	foundSet := make(map[string]bool)
	for i := 0; i < limit; i++ {
		docName := foundDocs[i]
		foundSet[docName] = true
		foundSet[strings.TrimSuffix(docName, ".md")] = true
		foundSet[filepath.Base(docName)] = true
	}

	found := 0
	for _, exp := range expectedDocs {
		if foundSet[exp] {
			found++
			continue
		}
		if foundSet[strings.TrimSuffix(exp, ".md")] {
			found++
			continue
		}
		if foundSet[filepath.Base(exp)] {
			found++
			continue
		}
	}

	return float64(found) / float64(len(expectedDocs))
}

func calcMRR(foundDocs, expectedDocs []string) float64 {
	expectedSet := make(map[string]bool)
	for _, d := range expectedDocs {
		expectedSet[d] = true
		expectedSet[strings.TrimSuffix(d, ".md")] = true
		expectedSet[filepath.Base(d)] = true
	}

	for rank, doc := range foundDocs {
		if expectedSet[doc] {
			return 1.0 / float64(rank+1)
		}
		if expectedSet[strings.TrimSuffix(doc, ".md")] {
			return 1.0 / float64(rank+1)
		}
		if expectedSet[filepath.Base(doc)] {
			return 1.0 / float64(rank+1)
		}
	}
	return 0
}