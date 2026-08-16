package logger

import (
	"encoding/json"
	"os"
	"testing"
	"time"
)

func TestNewPipelineLog(t *testing.T) {  // проверяет создание нового лога
	requestID := "test-request-123"
	userID := "test-user"

	log := NewPipelineLog(requestID, userID)

	if log.RequestID != requestID {
		t.Errorf("RequestID = %s, ожидалось %s", log.RequestID, requestID)
	}
	if log.UserID != userID {
		t.Errorf("UserID = %s, ожидалось %s", log.UserID, userID)
	}
	if log.Timestamp == "" {
		t.Error("Timestamp не должен быть пустым")
	}
	if log.Success {
		t.Error("Success должен быть false по умолчанию")
	}
}

func TestPipelineLogStruct(t *testing.T) {   // проверяет структуру PipelineLog
	log := &PipelineLog{
		RequestID:   "123",
		UserID:      "user",
		Timestamp:   time.Now().Format(time.RFC3339),
		DurationMs:  1000,
		Success:     true,
		FinalAnswer: "Тестовый ответ",
		Sources:     []string{"doc1.pdf", "doc2.pdf"},
	}

	if log.RequestID != "123" {
		t.Errorf("RequestID = %s, ожидалось 123", log.RequestID)
	}
	if log.DurationMs != 1000 {
		t.Errorf("DurationMs = %d, ожидалось 1000", log.DurationMs)
	}
	if !log.Success {
		t.Error("Success должен быть true")
	}
	if len(log.Sources) != 2 {
		t.Errorf("Sources = %d, ожидалось 2", len(log.Sources))
	}
}

func TestValidationLogStruct(t *testing.T) {
	log := &ValidationLog{
		Status:     "valid",
		Reason:     "Все хорошо",
		DurationMs: 50,
	}

	if log.Status != "valid" {
		t.Errorf("Status = %s, ожидалось valid", log.Status)
	}
	if log.Reason != "Все хорошо" {
		t.Errorf("Reason = %s, ожидалось 'Все хорошо'", log.Reason)
	}
	if log.DurationMs != 50 {
		t.Errorf("DurationMs = %d, ожидалось 50", log.DurationMs)
	}
}

func TestRetrievalLogStruct(t *testing.T) {
	log := &RetrievalLog{
		VectorResults: 10,
		TextResults:   5,
		FusedResults:  8,
		DurationMs:    100,
		FromCache:     false,
	}

	if log.VectorResults != 10 {
		t.Errorf("VectorResults = %d, ожидалось 10", log.VectorResults)
	}
	if log.TextResults != 5 {
		t.Errorf("TextResults = %d, ожидалось 5", log.TextResults)
	}
	if log.FusedResults != 8 {
		t.Errorf("FusedResults = %d, ожидалось 8", log.FusedResults)
	}
	if log.FromCache {
		t.Error("FromCache должен быть false")
	}
}

func TestRerankLogStruct(t *testing.T) {
	log := &RerankLog{
		BeforeCount: 10,
		AfterCount:  5,
		DurationMs:  50,
	}

	if log.BeforeCount != 10 {
		t.Errorf("BeforeCount = %d, ожидалось 10", log.BeforeCount)
	}
	if log.AfterCount != 5 {
		t.Errorf("AfterCount = %d, ожидалось 5", log.AfterCount)
	}
	if log.DurationMs != 50 {
		t.Errorf("DurationMs = %d, ожидалось 50", log.DurationMs)
	}
}

func TestLLMLogStruct(t *testing.T) {
	log := &LLMLog{
		Provider:   "openrouter",
		Model:      "gpt-4",
		ChunksSent: 5,
		TokensUsed: 1000,
		DurationMs: 500,
		Response:   "Тестовый ответ",
	}

	if log.Provider != "openrouter" {
		t.Errorf("Provider = %s, ожидалось openrouter", log.Provider)
	}
	if log.TokensUsed != 1000 {
		t.Errorf("TokensUsed = %d, ожидалось 1000", log.TokensUsed)
	}
	if log.Response != "Тестовый ответ" {
		t.Errorf("Response = %s, ожидалось 'Тестовый ответ'", log.Response)
	}
}

func TestPostProcessingLogStruct(t *testing.T) {
	log := &PostProcessingLog{
		CitationCount:  3,
		ValidCitations: 2,
		Hallucinations: 1,
		Verified:       true,
		DurationMs:     50,
	}

	if log.CitationCount != 3 {
		t.Errorf("CitationCount = %d, ожидалось 3", log.CitationCount)
	}
	if log.ValidCitations != 2 {
		t.Errorf("ValidCitations = %d, ожидалось 2", log.ValidCitations)
	}
	if log.Hallucinations != 1 {
		t.Errorf("Hallucinations = %d, ожидалось 1", log.Hallucinations)
	}
	if !log.Verified {
		t.Error("Verified должен быть true")
	}
}

func TestQueryProcessingLogStruct(t *testing.T) {
	log := &QueryProcessingLog{
		OriginalQuery:  "Что такое RAG?",
		RewrittenQuery: "RAG описание",
		HyDEQuery:      "RAG это технология",
		MultiQueries:   []string{"RAG", "Retrieval-Augmented Generation"},
		DurationMs:     100,
	}

	if log.OriginalQuery != "Что такое RAG?" {
		t.Errorf("OriginalQuery = %s, ожидалось 'Что такое RAG?'", log.OriginalQuery)
	}
	if log.RewrittenQuery != "RAG описание" {
		t.Errorf("RewrittenQuery = %s, ожидалось 'RAG описание'", log.RewrittenQuery)
	}
	if len(log.MultiQueries) != 2 {
		t.Errorf("MultiQueries = %d, ожидалось 2", len(log.MultiQueries))
	}
}

func TestPipelineLoggerSave(t *testing.T) {  // проверяет сохранение лога в файл
	testFile := "test_pipeline_logs.jsonl"
	defer os.Remove(testFile)

	logger := &PipelineLogger{
		filePath: testFile,
	}

	log := NewPipelineLog("test-123", "test-user")
	log.Success = true
	log.FinalAnswer = "Тестовый ответ"
	log.DurationMs = 100

	logger.Save(log)

	data, err := os.ReadFile(testFile)
	if err != nil {
		t.Errorf("Ошибка чтения файла логов: %v", err)
	}

	if len(data) == 0 {
		t.Error("Файл логов пуст")
	}

	var savedLog PipelineLog
	err = json.Unmarshal(data, &savedLog)
	if err != nil {
		t.Errorf("Ошибка парсинга JSON: %v", err)
	}

	if savedLog.RequestID != "test-123" {
		t.Errorf("RequestID = %s, ожидалось test-123", savedLog.RequestID)
	}
	if savedLog.UserID != "test-user" {
		t.Errorf("UserID = %s, ожидалось test-user", savedLog.UserID)
	}
	if savedLog.DurationMs != 100 {
		t.Errorf("DurationMs = %d, ожидалось 100", savedLog.DurationMs)
	}
}

func TestPipelineLoggerSaveMultiple(t *testing.T) { // проверяет сохранение нескольких логов
	testFile := "test_multiple_logs.jsonl"
	defer os.Remove(testFile)

	logger := &PipelineLogger{
		filePath: testFile,
	}

	for i := 0; i < 3; i++ {
		log := NewPipelineLog("req-"+string(rune('A'+i)), "user")
		log.DurationMs = int64((i + 1) * 100)
		logger.Save(log)
	}

	data, err := os.ReadFile(testFile)
	if err != nil {
		t.Errorf("Ошибка чтения файла: %v", err)
	}
	lines := 0
	for _, line := range data {
		if line == '\n' {
			lines++
		}
	}
	if lines != 3 {
		t.Errorf("Количество строк = %d, ожидалось 3", lines)
	}
}

func TestPipelineLoggerGetInstance(t *testing.T) {
	logger1 := GetPipelineLogger()
	logger2 := GetPipelineLogger()

	if logger1 != logger2 {
		t.Error("GetPipelineLogger() вернул разные экземпляры")
	}
}

func TestLogAnalyzerNew(t *testing.T) {  // проверяет создание анализатора логов
	testFile := "test_analyzer_new.jsonl"
	defer os.Remove(testFile)

	logger := &PipelineLogger{filePath: testFile}
	log := NewPipelineLog("test", "user")
	logger.Save(log)

	analyzer, err := NewLogAnalyzer(testFile)
	if err != nil {
		t.Errorf("NewLogAnalyzer() ошибка: %v", err)
	}

	if len(analyzer.logs) != 1 {
		t.Errorf("Анализатор загрузил %d логов, ожидалось 1", len(analyzer.logs))
	}
}

func TestLogAnalyzerSummary(t *testing.T) {
	testFile := "test_analyzer_summary.jsonl"
	defer os.Remove(testFile)

	logger := &PipelineLogger{filePath: testFile}

	logs := []*PipelineLog{
		{
			RequestID:  "1",
			UserID:     "user1",
			Timestamp:  time.Now().Format(time.RFC3339),
			DurationMs: 100,
			Success:    true,
			Retrieval: &RetrievalLog{
				VectorResults: 5,
				DurationMs:    10,
				FromCache:     false,
			},
			LLM: &LLMLog{
				TokensUsed: 100,
				DurationMs: 50,
			},
		},
		{
			RequestID:  "2",
			UserID:     "user2",
			Timestamp:  time.Now().Format(time.RFC3339),
			DurationMs: 200,
			Success:    false,
			Error:      "test error",
			Retrieval: &RetrievalLog{
				VectorResults: 3,
				DurationMs:    20,
				FromCache:     true,
			},
		},
	}

	for _, log := range logs {
		logger.Save(log)
	}

	analyzer, err := NewLogAnalyzer(testFile)
	if err != nil {
		t.Errorf("NewLogAnalyzer() ошибка: %v", err)
	}
	analyzer.Summary()
}
  
func TestLogAnalyzerWithEmptyFile(t *testing.T) {  // проверяет анализатор с пустым файлом
	testFile := "test_empty_analyzer.jsonl"
	defer os.Remove(testFile)

	os.WriteFile(testFile, []byte{}, 0644)

	analyzer, err := NewLogAnalyzer(testFile)
	if err != nil {
		t.Errorf("NewLogAnalyzer() с пустым файлом ошибка: %v", err)
	}

	if len(analyzer.logs) != 0 {
		t.Errorf("Анализатор загрузил %d логов, ожидалось 0", len(analyzer.logs))
	}
}

func TestLogAnalyzerWithInvalidJSON(t *testing.T) {
	testFile := "test_invalid_json.jsonl"
	defer os.Remove(testFile)

	os.WriteFile(testFile, []byte(`{"invalid": json}`), 0644)

	analyzer, err := NewLogAnalyzer(testFile)
	if err != nil {
		t.Errorf("NewLogAnalyzer() с невалидным JSON ошибка: %v", err)
	}

	if len(analyzer.logs) != 0 {
		t.Errorf("Анализатор загрузил %d логов из невалидного JSON, ожидалось 0", len(analyzer.logs))
	}
}

func TestLogAnalyzerPrintSlowest(t *testing.T) {
	testFile := "test_slowest.jsonl"
	defer os.Remove(testFile)

	logger := &PipelineLogger{filePath: testFile}

	for i := 0; i < 5; i++ {
		log := NewPipelineLog("req-"+string(rune('A'+i)), "user")
		log.DurationMs = int64((i + 1) * 100)
		logger.Save(log)
	}

	analyzer, err := NewLogAnalyzer(testFile)
	if err != nil {
		t.Errorf("NewLogAnalyzer() ошибка: %v", err)
	}

	analyzer.PrintSlowest(3)
}

func TestLogAnalyzerPrintSlowestWithEmptyLogs(t *testing.T) {
	testFile := "test_slowest_empty.jsonl"
	defer os.Remove(testFile)

	os.WriteFile(testFile, []byte{}, 0644)

	analyzer, err := NewLogAnalyzer(testFile)
	if err != nil {
		t.Errorf("NewLogAnalyzer() ошибка: %v", err)
	}
	analyzer.PrintSlowest(3)
}