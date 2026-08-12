package logger

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
)

type LogAnalyzer struct {  //  анализирует логи пайплайна
	logs []PipelineLog
}

func NewLogAnalyzer(filePath string) (*LogAnalyzer, error) {  // создает анализатор
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	analyzer := &LogAnalyzer{}
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		var log PipelineLog
		if err := json.Unmarshal([]byte(line), &log); err == nil {
			analyzer.logs = append(analyzer.logs, log)
		}
	}

	return analyzer, nil
}

func (a *LogAnalyzer) Summary() {
	if len(a.logs) == 0 {
		fmt.Println("Нет логов для анализа")
		return
	}

	var totalDuration, totalValidation, totalRetrieval, totalRerank, totalLLM, totalPost int64
	var successCount, failCount int
	var totalTokens int
	var maxDuration, minDuration int64
	var stagesDuration map[string]int64

	stagesDuration = make(map[string]int64)

	for _, log := range a.logs {
		totalDuration += log.DurationMs
		if log.Success {
			successCount++
		} else {
			failCount++
		}

		if log.Validation != nil {
			totalValidation += log.Validation.DurationMs
			stagesDuration["validation"] += log.Validation.DurationMs
		}
		if log.Retrieval != nil {
			totalRetrieval += log.Retrieval.DurationMs
			stagesDuration["retrieval"] += log.Retrieval.DurationMs
		}
		if log.Rerank != nil {
			totalRerank += log.Rerank.DurationMs
			stagesDuration["rerank"] += log.Rerank.DurationMs
		}
		if log.LLM != nil {
			totalLLM += log.LLM.DurationMs
			totalTokens += log.LLM.TokensUsed
			stagesDuration["llm"] += log.LLM.DurationMs
		}
		if log.PostProcessing != nil {
			totalPost += log.PostProcessing.DurationMs
			stagesDuration["post_processing"] += log.PostProcessing.DurationMs
		}

		if log.DurationMs > maxDuration {
			maxDuration = log.DurationMs
		}
		if minDuration == 0 || log.DurationMs < minDuration {
			minDuration = log.DurationMs
		}
	}

	count := len(a.logs)

	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("Анализ логов piplaine")
	fmt.Println(strings.Repeat("=", 60))

	fmt.Println("\n Общая статистика:")
	fmt.Printf("Всего запросов: %d\n", count)
	fmt.Printf("Успешных: %d (%.1f%%)\n", successCount, float64(successCount)/float64(count)*100)
	if failCount > 0 {
		fmt.Printf("Ошибок: %d (%.1f%%)\n", failCount, float64(failCount)/float64(count)*100)
	}
	fmt.Printf("Среднее время: %.2f сек\n", float64(totalDuration)/float64(count)/1000)
	fmt.Printf("Минимальное время: %.2f сек\n", float64(minDuration)/1000)
	fmt.Printf("Максимальное время: %.2f сек\n", float64(maxDuration)/1000)

	fmt.Println("\n Среднее время по этапам:")
	fmt.Printf("Валидация: %.2f сек\n", float64(totalValidation)/float64(count)/1000)
	fmt.Printf("Поиск: %.2f сек\n", float64(totalRetrieval)/float64(count)/1000)
	fmt.Printf("Реренкинг: %.2f сек\n", float64(totalRerank)/float64(count)/1000)
	fmt.Printf("LLM: %.2f сек\n", float64(totalLLM)/float64(count)/1000)
	fmt.Printf("Пост-обработка: %.2f сек\n", float64(totalPost)/float64(count)/1000)

	if totalTokens > 0 {
		fmt.Printf("\n Токены:\n")
		fmt.Printf("Всего токенов: %d\n", totalTokens)
		fmt.Printf("Среднее токенов на запрос: %d\n", totalTokens/count)
	}

	a.printRetrievalStats()
}

func (a *LogAnalyzer) printRetrievalStats() {  //  печатает статистику по поиску
	var totalVector, totalText, totalFused int
	var cacheHits, cacheMisses int

	for _, log := range a.logs {
		if log.Retrieval != nil {
			totalVector += log.Retrieval.VectorResults
			totalText += log.Retrieval.TextResults
			totalFused += log.Retrieval.FusedResults
			if log.Retrieval.FromCache {
				cacheHits++
			} else {
				cacheMisses++
			}
		}
	}

	count := len(a.logs)
	if count == 0 || totalVector == 0 {
		return
	}

	fmt.Println("\n Детали поиска:")
	fmt.Printf("Среднее векторных результатов: %d\n", totalVector/count)
	fmt.Printf("Среднее текстовых результатов: %d\n", totalText/count)
	fmt.Printf("Среднее объединённых результатов: %d\n", totalFused/count)
	fmt.Printf("Кэш: %d попаданий, %d промахов\n", cacheHits, cacheMisses)
}

func (a *LogAnalyzer) PrintSlowest(topN int) { // печатает самые медленные запросы
	if len(a.logs) == 0 {
		return
	}

	sorted := make([]PipelineLog, len(a.logs))
	copy(sorted, a.logs)

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].DurationMs > sorted[j].DurationMs
	})

	fmt.Println("\n Самые медленные запросы:")
	for i := 0; i < topN && i < len(sorted); i++ {
		log := sorted[i]
		fmt.Printf("   #%d: %.2f сек | %s | %s\n",
			i+1,
			float64(log.DurationMs)/1000,
			log.RequestID,
			log.UserID,
		)
	}
}