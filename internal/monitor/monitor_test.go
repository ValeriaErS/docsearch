package monitor

import (
	"testing"
	"time"
)

func TestMetricsStartNew(t *testing.T) {
	m := &Metrics{}
	m.StartNew("тестовый запрос", "testuser")

	if m.Query != "тестовый запрос" {
		t.Errorf("Query = %s, ожидалось 'тестовый запрос'", m.Query)
	}
	if m.UserID != "testuser" {
		t.Errorf("UserID = %s, ожидалось 'testuser'", m.UserID)
	}
	if !m.Success {
		t.Error("Success должен быть true")
	}
	if m.StartTime.IsZero() {
		t.Error("StartTime не должен быть нулевым")
	}
}

func TestMetricsEnd(t *testing.T) {
	m := &Metrics{}
	m.StartNew("test", "user")
	time.Sleep(10 * time.Millisecond)
	m.End("тестовый ответ")

	if m.Answer != "тестовый ответ" {
		t.Errorf("Answer = %s, ожидалось 'тестовый ответ'", m.Answer)
	}
	if m.TotalDurationMs < 5 {
		t.Errorf("TotalDurationMs = %d, ожидалось > 5", m.TotalDurationMs)
	}
	if m.EndTime.IsZero() {
		t.Error("EndTime не должен быть нулевым")
	}
}

func TestMetricsSetError(t *testing.T) {
	m := &Metrics{}
	m.SetError("тестовая ошибка")

	if m.Success {
		t.Error("Success должен быть false после SetError")
	}
	if m.Error != "тестовая ошибка" {
		t.Errorf("Error = %s, ожидалось 'тестовая ошибка'", m.Error)
	}
}

func TestMetricsSetDurations(t *testing.T) {
	m := &Metrics{}
	
	duration := 100 * time.Millisecond
	m.SetEmbeddingDuration(duration)
	m.SetSearchDuration(duration)
	m.SetLLMDuration(duration)
	m.SetRerankDuration(duration)

	if m.EmbeddingDurationMs != 100 {
		t.Errorf("EmbeddingDurationMs = %d, ожидалось 100", m.EmbeddingDurationMs)
	}
	if m.SearchDurationMs != 100 {
		t.Errorf("SearchDurationMs = %d, ожидалось 100", m.SearchDurationMs)
	}
	if m.LLMDurationMs != 100 {
		t.Errorf("LLMDurationMs = %d, ожидалось 100", m.LLMDurationMs)
	}
	if m.RerankDurationMs != 100 {
		t.Errorf("RerankDurationMs = %d, ожидалось 100", m.RerankDurationMs)
	}
}

func TestMetricsSetChunks(t *testing.T) {
	m := &Metrics{}
	
	m.SetChunksFound(10)
	m.SetChunksAfterRerank(5)
	m.SetChunksAfterCompression(3)

	if m.ChunksFound != 10 {
		t.Errorf("ChunksFound = %d, ожидалось 10", m.ChunksFound)
	}
	if m.ChunksAfterRerank != 5 {
		t.Errorf("ChunksAfterRerank = %d, ожидалось 5", m.ChunksAfterRerank)
	}
	if m.ChunksAfterCompression != 3 {
		t.Errorf("ChunksAfterCompression = %d, ожидалось 3", m.ChunksAfterCompression)
	}
}

func TestMetricsSetTokensAndModel(t *testing.T) {
	m := &Metrics{}
	
	m.SetTokensUsed(1000)
	m.SetModel("test-model")

	if m.TokensUsed != 1000 {
		t.Errorf("TokensUsed = %d, ожидалось 1000", m.TokensUsed)
	}
	if m.Model != "test-model" {
		t.Errorf("Model = %s, ожидалось 'test-model'", m.Model)
	}
}

func TestMetricsSetSources(t *testing.T) {
	m := &Metrics{}
	
	sources := []string{"doc1.pdf", "doc2.pdf", "doc3.pdf"}
	m.SetSources(sources)

	if len(m.Sources) != 3 {
		t.Errorf("Sources length = %d, ожидалось 3", len(m.Sources))
	}
	if m.Sources[0] != "doc1.pdf" {
		t.Errorf("Sources[0] = %s, ожидалось 'doc1.pdf'", m.Sources[0])
	}
}

func TestMetricsSetRewrittenQuery(t *testing.T) {
	m := &Metrics{}
	
	m.SetRewrittenQuery("переписанный запрос")

	if m.RewrittenQuery != "переписанный запрос" {
		t.Errorf("RewrittenQuery = %s, ожидалось 'переписанный запрос'", m.RewrittenQuery)
	}
}

func TestMetricsSetValidation(t *testing.T) {
	m := &Metrics{}
	
	m.Validated = true
	m.ValidationReason = "все хорошо"
	m.ValidationDurationMs = 50

	if !m.Validated {
		t.Error("Validated должен быть true")
	}
	if m.ValidationReason != "все хорошо" {
		t.Errorf("ValidationReason = %s, ожидалось 'все хорошо'", m.ValidationReason)
	}
	if m.ValidationDurationMs != 50 {
		t.Errorf("ValidationDurationMs = %d, ожидалось 50", m.ValidationDurationMs)
	}
}

func TestMetricsSetAdaptiveFields(t *testing.T) {
	m := &Metrics{}
	
	m.SetQueryComplexity("complex")
	m.SetRetrievalStrategy("hybrid")
	m.SetRetrievalRounds(3)

	if m.QueryComplexity != "complex" {
		t.Errorf("QueryComplexity = %s, ожидалось 'complex'", m.QueryComplexity)
	}
	if m.RetrievalStrategy != "hybrid" {
		t.Errorf("RetrievalStrategy = %s, ожидалось 'hybrid'", m.RetrievalStrategy)
	}
	if m.RetrievalRounds != 3 {
		t.Errorf("RetrievalRounds = %d, ожидалось 3", m.RetrievalRounds)
	}
}

func TestMetricCollectorGetInstance(t *testing.T) {
	collector1 := GetCollector()
	collector2 := GetCollector()

	if collector1 != collector2 {
		t.Error("GetCollector() вернул разные экземпляры")
	}
}

func TestMetricCollectorSave(t *testing.T) {
	collector := GetCollector()
	
	m := Metrics{}
	m.StartNew("тестовый запрос", "testuser")
	m.End("тестовый ответ")
	
	collector.Save(m)
	
	metrics := collector.GetMetrics()
	found := false
	for _, metric := range metrics {
		if metric.Query == "тестовый запрос" && metric.UserID == "testuser" {
			found = true
			break
		}
	}
	if !found {
		t.Error("MetricCollector не сохранил метрику")
	}
}

func TestMetricCollectorGetMetrics(t *testing.T) {
	collector := GetCollector()
	
	metrics := collector.GetMetrics()
	if metrics == nil {
		t.Error("GetMetrics() вернул nil")
	}
}

func TestMetricCollectorPrintSummary(t *testing.T) {
	collector := GetCollector()
	
	m := Metrics{}
	m.StartNew("test", "user")
	m.End("answer")
	m.SetTokensUsed(100)
	collector.Save(m)
	
	collector.PrintSummary()
}

func TestMetricCollectorExportSummary(t *testing.T) {
	collector := GetCollector()
	
	m := Metrics{}
	m.StartNew("test", "user")
	m.End("answer")
	collector.Save(m)
	
	collector.ExportSummary()
}

func TestMetricCollectorLoad(t *testing.T) {
	collector := GetCollector()
	
	m := Metrics{}
	m.StartNew("test", "user")
	m.End("answer")
	collector.Save(m)
	
	collector.Load()
}

func TestMetricsAllFields(t *testing.T) {
	m := &Metrics{}
	
	m.StartNew("query", "user")
	m.SetModel("model")
	m.SetTokensUsed(500)
	m.SetSources([]string{"doc1.pdf"})
	m.SetRewrittenQuery("rewritten")
	m.SetQueryComplexity("medium")
	m.SetRetrievalStrategy("hybrid")
	m.SetRetrievalRounds(2)
	m.SetEmbeddingDuration(50 * time.Millisecond)
	m.SetSearchDuration(100 * time.Millisecond)
	m.SetLLMDuration(200 * time.Millisecond)
	m.SetRerankDuration(30 * time.Millisecond)
	m.SetChunksFound(10)
	m.SetChunksAfterRerank(5)
	m.SetChunksAfterCompression(3)
	m.End("answer")

	if m.Query != "query" {
		t.Errorf("Query = %s, ожидалось 'query'", m.Query)
	}
	if m.UserID != "user" {
		t.Errorf("UserID = %s, ожидалось 'user'", m.UserID)
	}
	if m.Model != "model" {
		t.Errorf("Model = %s, ожидалось 'model'", m.Model)
	}
	if m.TokensUsed != 500 {
		t.Errorf("TokensUsed = %d, ожидалось 500", m.TokensUsed)
	}
	if m.Answer != "answer" {
		t.Errorf("Answer = %s, ожидалось 'answer'", m.Answer)
	}
}