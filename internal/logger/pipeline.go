package logger

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

type PipelineLog struct {   // полный лог одного запроса
	RequestID   string                 `json:"request_id"`
	UserID      string                 `json:"user_id"`
	Timestamp   string                 `json:"timestamp"`
	DurationMs  int64                  `json:"duration_ms"`
	Success     bool                   `json:"success"`
	Error       string                 `json:"error,omitempty"`

	Validation *ValidationLog          `json:"validation,omitempty"`   // валидация

	QueryProcessing *QueryProcessingLog `json:"query_processing,omitempty"`  // обработка запроса

	Retrieval *RetrievalLog            `json:"retrieval,omitempty"`  // поиск

	Rerank *RerankLog                  `json:"rerank,omitempty"`  // реренкинг

	LLM *LLMLog                        `json:"llm,omitempty"`

	PostProcessing *PostProcessingLog  `json:"post_processing,omitempty"`  // пост обработка

	FinalAnswer string                 `json:"final_answer,omitempty"`
	Sources     []string               `json:"sources,omitempty"`
}

type ValidationLog struct {  
	Status string  `json:"status"`
	Reason string  `json:"reason,omitempty"`
	DurationMs int64 `json:"duration_ms"`
}

type QueryProcessingLog struct {  
	OriginalQuery    string   `json:"original_query"`
	RewrittenQuery   string   `json:"rewritten_query,omitempty"`
	HyDEQuery        string   `json:"hyde_query,omitempty"`
	MultiQueries     []string `json:"multi_queries,omitempty"`
	DurationMs       int64    `json:"duration_ms"`
}

type RetrievalLog struct {   
	VectorResults int      `json:"vector_results"`
	TextResults   int      `json:"text_results"`
	FusedResults  int      `json:"fused_results"`
	TopScores     []float64 `json:"top_scores,omitempty"`
	TopDocs       []string `json:"top_docs,omitempty"`
	DurationMs    int64    `json:"duration_ms"`
	FromCache     bool     `json:"from_cache"`
}

type RerankLog struct {  
	BeforeCount  int      `json:"before_count"`
	AfterCount   int      `json:"after_count"`
	TopScores    []float64 `json:"top_scores,omitempty"`
	DurationMs   int64    `json:"duration_ms"`
}

type LLMLog struct {
	Provider     string `json:"provider"`
	Model        string `json:"model"`
	ChunksSent   int    `json:"chunks_sent"`
	TokensUsed   int    `json:"tokens_used"`
	DurationMs   int64  `json:"duration_ms"`
	Response     string `json:"response,omitempty"`
}

type PostProcessingLog struct {
	CitationCount   int  `json:"citation_count"`
	ValidCitations  int  `json:"valid_citations"`
	Hallucinations  int  `json:"hallucinations"`
	Verified        bool `json:"verified"`
	DurationMs     int64 `json:"duration_ms"`
}

type PipelineLogger struct {
	mu       sync.Mutex
	filePath string
}

var pipelineLogger *PipelineLogger
var loggerOnce sync.Once

func GetPipelineLogger() *PipelineLogger {
	loggerOnce.Do(func() {
		pipelineLogger = &PipelineLogger{
			filePath: "./pipeline_logs.jsonl",
		}
	})
	return pipelineLogger
}

func (l *PipelineLogger) Save(log *PipelineLog) {
	l.mu.Lock()
	defer l.mu.Unlock()

	data, err := json.Marshal(log)
	if err != nil {
		fmt.Printf("Ошибка маршалинга лога: %v\n", err)
		return
	}

	f, err := os.OpenFile(l.filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("Ошибка открытия файла логов: %v\n", err)
		return
	}
	defer f.Close()

	f.Write(data)
	f.Write([]byte("\n"))
}

func NewPipelineLog(requestID, userID string) *PipelineLog {  //  создает новый лог с request_id
	return &PipelineLog{
		RequestID: requestID,
		UserID:    userID,
		Timestamp: time.Now().Format(time.RFC3339),
	}
}