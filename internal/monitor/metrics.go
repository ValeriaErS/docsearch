package monitor

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
	"strings"
)

type Metrics struct {  //  хранит метрики одного запроса
	mu sync.Mutex

	StartTime        time.Time `json:"start_time"`  // dременные метки
	EndTime          time.Time `json:"end_time"`
	TotalDurationMs  int64     `json:"total_duration_ms"`

	EmbeddingDurationMs int64 `json:"embedding_duration_ms"` // этапы
	SearchDurationMs    int64 `json:"search_duration_ms"`
	RerankDurationMs    int64 `json:"rerank_duration_ms"`
	LLMDurationMs       int64 `json:"llm_duration_ms"`

	ChunksFound     int `json:"chunks_found"` // количество
	ChunksAfterRerank int `json:"chunks_after_rerank"`
	ChunksAfterCompression int `json:"chunks_after_compression"`
	TokensUsed      int `json:"tokens_used"`


	Query       string   `json:"query"` // запрос и ответ
	RewrittenQuery string `json:"rewritten_query,omitempty"`
	Answer      string   `json:"answer,omitempty"`
	Sources     []string `json:"sources"`
	Model       string   `json:"model"`

	Success bool   `json:"success"`  // статус
	Error   string `json:"error,omitempty"`

	UserID string `json:"user_id"`
}

type MetricCollector struct {  //  собирает метрики
	mu       sync.Mutex
	metrics  []Metrics
	filePath string
}

var (
	collector *MetricCollector
	once      sync.Once
)

func GetCollector() *MetricCollector {  //  возвращает синглтон коллектора
	once.Do(func() {
		collector = &MetricCollector{
			metrics:  []Metrics{},
			filePath: "./metrics.jsonl",
		}
		collector.Load()
	})
	return collector
}

func (m *Metrics) StartNew(query, userID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.StartTime = time.Now()
	m.Query = query
	m.UserID = userID
	m.Success = true
}

func (m *Metrics) End(answer string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.EndTime = time.Now()
	m.TotalDurationMs = m.EndTime.Sub(m.StartTime).Milliseconds()
	m.Answer = answer
}

func (m *Metrics) SetError(err string) {  //  устанавливает ошибку
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Success = false
	m.Error = err
}

func (m *Metrics) SetEmbeddingDuration(duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.EmbeddingDurationMs = duration.Milliseconds()
}

func (m *Metrics) SetSearchDuration(duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.SearchDurationMs = duration.Milliseconds()
}

func (m *Metrics) SetLLMDuration(duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.LLMDurationMs = duration.Milliseconds()
}

func (m *Metrics) SetRerankDuration(duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.RerankDurationMs = duration.Milliseconds()
}

func (m *Metrics) SetChunksFound(count int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ChunksFound = count
}

func (m *Metrics) SetChunksAfterRerank(count int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ChunksAfterRerank = count
}

func (m *Metrics) SetChunksAfterCompression(count int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ChunksAfterCompression = count
}

func (m *Metrics) SetTokensUsed(count int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.TokensUsed = count
}

func (m *Metrics) SetSources(sources []string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Sources = sources
}

func (m *Metrics) SetModel(model string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Model = model
}

func (m *Metrics) SetRewrittenQuery(query string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.RewrittenQuery = query
}

func (c *MetricCollector) Save(m Metrics) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.metrics = append(c.metrics, m)

	data, err := json.Marshal(m)  // сохр в JSONL файл
	if err != nil {
		fmt.Printf("Ошибка маршалинга метрик: %v\n", err)
		return
	}

	f, err := os.OpenFile(c.filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("Ошибка открытия файла метрик: %v\n", err)
		return
	}
	defer f.Close()

	f.Write(data)
	f.Write([]byte("\n"))
}

func (c *MetricCollector) GetMetrics() []Metrics {  //  возвращает все метрики
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.metrics
}

func (c *MetricCollector) PrintSummary() {  //  печатает сводку по метрикам
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.metrics) == 0 {
		fmt.Println("Нет метрик")
		return
	}

	var totalDuration, totalEmbedding, totalSearch, totalLLM int64
	var totalTokens int
	var successCount int

	for _, m := range c.metrics {
		totalDuration += m.TotalDurationMs
		totalEmbedding += m.EmbeddingDurationMs
		totalSearch += m.SearchDurationMs
		totalLLM += m.LLMDurationMs
		totalTokens += m.TokensUsed
		if m.Success {
			successCount++
		}
	}

	count := len(c.metrics)
	fmt.Println("\n Сводка метрик:")
	fmt.Printf("Всего запросов: %d\n", count)
	fmt.Printf("Успешных: %d (%.1f%%)\n", successCount, float64(successCount)/float64(count)*100)
	fmt.Printf("Среднее время: %.2f сек\n", float64(totalDuration)/float64(count)/1000)
	fmt.Printf("Среднее время эмбеддинга: %.2f сек\n", float64(totalEmbedding)/float64(count)/1000)
	fmt.Printf("Среднее время поиска: %.2f сек\n", float64(totalSearch)/float64(count)/1000)
	fmt.Printf("Среднее время LLM: %.2f сек\n", float64(totalLLM)/float64(count)/1000)
	fmt.Printf("Среднее токенов: %d\n", totalTokens/count)
}
// ExportSummary печатает сводку по метрикам
func (c *MetricCollector) ExportSummary() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.metrics) == 0 {
		fmt.Println("📊 Нет метрик")
		return
	}

	var totalDuration, totalEmbedding, totalSearch, totalLLM int64
	var totalTokens int
	var successCount int

	for _, m := range c.metrics {
		totalDuration += m.TotalDurationMs
		totalEmbedding += m.EmbeddingDurationMs
		totalSearch += m.SearchDurationMs
		totalLLM += m.LLMDurationMs
		totalTokens += m.TokensUsed
		if m.Success {
			successCount++
		}
	}

	count := len(c.metrics)
	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("📊 СВОДКА МЕТРИК")
	fmt.Println(strings.Repeat("=", 50))
	fmt.Printf("   Всего запросов: %d\n", count)
	fmt.Printf("   Успешных: %d (%.1f%%)\n", successCount, float64(successCount)/float64(count)*100)
	fmt.Printf("   Среднее время: %.2f сек\n", float64(totalDuration)/float64(count)/1000)
	fmt.Printf("   Среднее время эмбеддинга: %.2f сек\n", float64(totalEmbedding)/float64(count)/1000)
	fmt.Printf("   Среднее время поиска: %.2f сек\n", float64(totalSearch)/float64(count)/1000)
	fmt.Printf("   Среднее время LLM: %.2f сек\n", float64(totalLLM)/float64(count)/1000)
	fmt.Printf("   Среднее токенов: %d\n", totalTokens/count)
	fmt.Println(strings.Repeat("=", 50))
}
// Load загружает метрики из файла
func (c *MetricCollector) Load() {
	c.mu.Lock()
	defer c.mu.Unlock()

	data, err := os.ReadFile(c.filePath)
	if err != nil {
		return
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		var m Metrics
		if err := json.Unmarshal([]byte(line), &m); err == nil {
			c.metrics = append(c.metrics, m)
		}
	}
	fmt.Printf("📊 Загружено %d метрик из файла\n", len(c.metrics))
}