package eval

import (
    "bufio"
    "encoding/json"
    "fmt"
    "os"
    "strings"
)
type GoldenQuestion struct {
    Query          string   `json:"query"`
    RelevantDocs   []string `json:"relevant_docs"`
    RelevantChunks []string `json:"relevant_chunks"`
}
type GoldenResult struct {
    Query           string   `json:"query"`
    FoundDocs       []string `json:"found_docs"`
    RecallDocs      float64  `json:"recall_docs"`
    Success         bool     `json:"success"`
    MissedDocs      []string `json:"missed_docs"`
}
type GoldenSummary struct {
    TotalQueries    int     `json:"total_queries"`
    SuccessCount    int     `json:"success_count"`
    SuccessRate     float64 `json:"success_rate"`
    AvgRecallDocs   float64 `json:"avg_recall_docs"`
}
func LoadGoldenQuestions(path string) ([]GoldenQuestion, error) {  // загружает вопросы из файла
    file, err := os.Open(path)
    if err != nil {
        return nil, err
    }
    defer file.Close()

    var questions []GoldenQuestion
    scanner := bufio.NewScanner(file)
    for scanner.Scan() {
        line := strings.TrimSpace(scanner.Text())
        if line == "" {
            continue
        }
        var q GoldenQuestion
        if err := json.Unmarshal([]byte(line), &q); err != nil {
            continue
        }
        questions = append(questions, q)
    }
    return questions, scanner.Err()
}

func EvaluateGolden(questions []GoldenQuestion, foundDocsFunc func(string) []string) GoldenSummary { //проверяет систему на золотом наборе
    var results []GoldenResult
    totalRecall := 0.0
    successCount := 0

    for _, q := range questions {
        foundDocs := foundDocsFunc(q.Query)

        foundSet := make(map[string]bool) // считаю Recall по документам
        for _, d := range foundDocs {
            foundSet[d] = true
        }

        found := 0
        missed := []string{}
        for _, expected := range q.RelevantDocs {
            if foundSet[expected] {
                found++
            } else {
                missed = append(missed, expected)
            }
        }

        recall := 0.0
        if len(q.RelevantDocs) > 0 {
            recall = float64(found) / float64(len(q.RelevantDocs))
        }

        success := recall >= 0.5

        if success {
            successCount++
        }
        totalRecall += recall

        results = append(results, GoldenResult{
            Query:      q.Query,
            FoundDocs:  foundDocs,
            RecallDocs: recall,
            Success:    success,
            MissedDocs: missed,
        })
    }

    return GoldenSummary{
        TotalQueries:  len(questions),
        SuccessCount:  successCount,
        SuccessRate:   float64(successCount) / float64(len(questions)) * 100,
        AvgRecallDocs: totalRecall / float64(len(questions)),
    }
}

func PrintGoldenSummary(summary GoldenSummary) {
    fmt.Println("\n" + strings.Repeat("=", 50))
    fmt.Println("Золотой набор")
    fmt.Println(strings.Repeat("=", 50))
    fmt.Printf("Всего вопросов: %d\n", summary.TotalQueries)
    fmt.Printf("Успешных ответов: %d (%.1f%%)\n", summary.SuccessCount, summary.SuccessRate)
    fmt.Printf("Средний Recall: %.1f%%\n", summary.AvgRecallDocs*100)
    fmt.Println(strings.Repeat("=", 50))
}