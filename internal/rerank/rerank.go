package rerank

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type Reranker struct {
	apiKey string
	client *http.Client
}

type RerankRequest struct {
	Model     string   `json:"model"`
	Query     string   `json:"query"`
	Documents []string `json:"documents"`
	TopN      int      `json:"top_n"`
}

type RerankResult struct {
	Index int     `json:"index"`
	Score float64 `json:"relevance_score"`
}

type RerankResponse struct {
	Results []RerankResult `json:"results"`
}

func NewReranker() *Reranker {  // создает новый реранкер
	return &Reranker{
		apiKey: os.Getenv("COHERE_API_KEY"),
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (r *Reranker) Rerank(query string, documents []string, topK int) ([]int, []float64, error) {  //  выполняет реранкинг документов через Cohere API
	if len(documents) == 0 {
		return []int{}, []float64{}, nil
	}

	if r.apiKey == "" {
		fmt.Println("COHERE_API_KEY не задан, пропускаем реранкинг")
		return nil, nil, nil
	}

	if len(documents) > 50 {  // Cohere ограничивает 50 документов
		documents = documents[:50]
	}

	reqBody := RerankRequest{
		Model:     "rerank-english-v3.0",
		Query:     query,
		Documents: documents,
		TopN:      topK,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, nil, fmt.Errorf("ошибка маршалинга: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.cohere.ai/v1/rerank", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, nil, fmt.Errorf("ошибка создания запроса: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+r.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.client.Do(req)
	if err != nil {
		fmt.Printf("Ошибка Cohere: %v, пропускаем реранкинг\n", err)
		return nil, nil, nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("ошибка чтения ответа: %w", err)
	}

	if resp.StatusCode != 200 {
		fmt.Printf("Cohere ошибка %d: %s, пропускаем реранкинг\n", resp.StatusCode, string(body))
		return nil, nil, nil
	}

	var result RerankResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, nil, fmt.Errorf("ошибка парсинга: %w", err)
	}

	indices := []int{}
	scores := []float64{}
	for _, r := range result.Results {
		indices = append(indices, r.Index)
		scores = append(scores, r.Score)
	}

	fmt.Printf("Реренкинг: %d → %d документов\n", len(documents), len(indices))
	return indices, scores, nil
}