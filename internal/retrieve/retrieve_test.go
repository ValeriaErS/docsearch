package retrieve

import (
	"math"
	"testing"
)

func TestSimilarity(t *testing.T) {   // проверяет косинусное сходство векторов
	tests := []struct {
		name     string
		a        []float64
		b        []float64
		expected float64
	}{
		{
			name:     "одинаковые векторы",
			a:        []float64{1, 0, 0},
			b:        []float64{1, 0, 0},
			expected: 1.0,
		},
		{
			name:     "ортогональные векторы",
			a:        []float64{1, 0, 0},
			b:        []float64{0, 1, 0},
			expected: 0.0,
		},
		{
			name:     "противоположные векторы",
			a:        []float64{1, 0, 0},
			b:        []float64{-1, 0, 0},
			expected: -1.0,
		},
		{
			name:     "разная длина",
			a:        []float64{1, 0},
			b:        []float64{1, 0, 0},
			expected: 0.0,
		},
		{
			name:     "пустые векторы",
			a:        []float64{},
			b:        []float64{},
			expected: 0.0,
		},
		{
			name:     "векторы с отрицательными значениями",
			a:        []float64{-1, -2},
			b:        []float64{-1, -2},
			expected: 1.0,
		},
		{
			name:     "векторы с дробями",
			a:        []float64{0.5, 0.5},
			b:        []float64{1.0, 0.0},
			expected: 0.70710678,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := similarity(tt.a, tt.b)
			if math.Abs(result-tt.expected) > 0.0001 {
				t.Errorf("similarity() = %v, ожидалось %v", result, tt.expected)
			}
		})
	}
}

func TestSearch(t *testing.T) {         // проверяет поиск по векторам
	texts := []string{"doc1", "doc2", "doc3", "doc4", "doc5"}
	docs := []string{"doc1.txt", "doc2.txt", "doc3.txt", "doc4.txt", "doc5.txt"}
	vectors := [][]float64{
		{1, 0, 0},
		{0, 1, 0},
		{0, 0, 1},
		{0.5, 0.5, 0},
		{0, 0.5, 0.5},
	}
	query := []float64{1, 0.5, 0}

	topK := 3
	resultTexts, _, _ := Search(texts, docs, vectors, query, topK)

	if len(resultTexts) != topK {
		t.Errorf("Search() вернул %d результатов, ожидалось %d", len(resultTexts), topK)
	}

	expected := map[string]bool{"doc1": true, "doc2": true, "doc4": true}
	for _, doc := range resultTexts {
		if !expected[doc] {
			t.Errorf("Search() вернул неожиданный документ: %s", doc)
		}
		delete(expected, doc)
	}
	if len(expected) > 0 {
		t.Errorf("Search() не вернул ожидаемые документы: %v", expected)
	}
}

func TestSearchEmpty(t *testing.T) {  // проверяет поиск с пустыми данными
	texts := []string{}
	docs := []string{}
	vectors := [][]float64{}
	query := []float64{1, 0, 0}

	resultTexts, resultDocs, resultScores := Search(texts, docs, vectors, query, 5)

	if len(resultTexts) != 0 {
		t.Errorf("Search() с пустыми векторами вернул %d результатов, ожидалось 0", len(resultTexts))
	}
	if len(resultDocs) != 0 {
		t.Errorf("Search() с пустыми векторами вернул %d документов, ожидалось 0", len(resultDocs))
	}
	if len(resultScores) != 0 {
		t.Errorf("Search() с пустыми векторами вернул %d оценок, ожидалось 0", len(resultScores))
	}
}

func TestSearchTopKZero(t *testing.T) {  // проверяет поиск с TopK = 0
	texts := []string{"doc1", "doc2"}
	docs := []string{"doc1.txt", "doc2.txt"}
	vectors := [][]float64{{1, 0}, {0, 1}}
	query := []float64{1, 0}

	resultTexts, resultDocs, resultScores := Search(texts, docs, vectors, query, 0)

	if len(resultTexts) != 0 {
		t.Errorf("Search() с TopK=0 вернул %d результатов, ожидалось 0", len(resultTexts))
	}
	if len(resultDocs) != 0 {
		t.Errorf("Search() с TopK=0 вернул %d документов, ожидалось 0", len(resultDocs))
	}
	if len(resultScores) != 0 {
		t.Errorf("Search() с TopK=0 вернул %d оценок, ожидалось 0", len(resultScores))
	}
}

func TestSearchSingleVector(t *testing.T) {  // проверяет поиск с одним вектором
	texts := []string{"doc1"}
	docs := []string{"doc1.txt"}
	vectors := [][]float64{{1, 0, 0}}
	query := []float64{1, 0, 0}

	resultTexts, _, _ := Search(texts, docs, vectors, query, 5)

	if len(resultTexts) != 1 {
		t.Errorf("Search() с одним вектором вернул %d результатов, ожидалось 1", len(resultTexts))
	}
	if resultTexts[0] != "doc1" {
		t.Errorf("Search() результат = %s, ожидался doc1", resultTexts[0])
	}
}

func TestReciprocalRankFusion(t *testing.T) {
	list1 := []map[string]interface{}{
		{"id": "doc1", "score": 0.9, "payload": map[string]interface{}{"text": "doc1"}},
		{"id": "doc2", "score": 0.8, "payload": map[string]interface{}{"text": "doc2"}},
		{"id": "doc3", "score": 0.7, "payload": map[string]interface{}{"text": "doc3"}},
	}
	list2 := []map[string]interface{}{
		{"id": "doc2", "score": 0.85, "payload": map[string]interface{}{"text": "doc2"}},
		{"id": "doc3", "score": 0.75, "payload": map[string]interface{}{"text": "doc3"}},
		{"id": "doc4", "score": 0.65, "payload": map[string]interface{}{"text": "doc4"}},
	}

	result := ReciprocalRankFusion(list1, list2)

	if len(result) == 0 {
		t.Error("ReciprocalRankFusion() вернул пустой результат")
	}

	if result[0]["id"] != "doc2" {
		t.Errorf("ReciprocalRankFusion() первый результат = %v, ожидался doc2", result[0]["id"])
	}
}

func TestReciprocalRankFusionEmpty(t *testing.T) {  // проверяет RRF с пустыми списками
	list1 := []map[string]interface{}{}
	list2 := []map[string]interface{}{}

	result := ReciprocalRankFusion(list1, list2)

	if len(result) != 0 {
		t.Errorf("ReciprocalRankFusion() с пустыми списками вернул %d результатов, ожидалось 0", len(result))
	}
}

func TestReciprocalRankFusionSingleList(t *testing.T) {
	list1 := []map[string]interface{}{
		{"id": "doc1", "score": 0.9, "payload": map[string]interface{}{"text": "doc1"}},
		{"id": "doc2", "score": 0.8, "payload": map[string]interface{}{"text": "doc2"}},
	}

	result := ReciprocalRankFusion(list1)

	if len(result) != 2 {
		t.Errorf("ReciprocalRankFusion() с одним списком вернул %d результатов, ожидалось 2", len(result))
	}
	if result[0]["id"] != "doc1" {
		t.Errorf("ReciprocalRankFusion() первый результат = %v, ожидался doc1", result[0]["id"])
	}
}

func TestReciprocalRankFusionDuplicateIDs(t *testing.T) {
	list1 := []map[string]interface{}{
		{"id": "doc1", "score": 0.9, "payload": map[string]interface{}{"text": "doc1"}},
		{"id": "doc1", "score": 0.8, "payload": map[string]interface{}{"text": "doc1"}},
	}
	list2 := []map[string]interface{}{
		{"id": "doc1", "score": 0.85, "payload": map[string]interface{}{"text": "doc1"}},
	}

	result := ReciprocalRankFusion(list1, list2)

	if len(result) != 1 {
		t.Errorf("ReciprocalRankFusion() с дубликатами вернул %d результатов, ожидалось 1", len(result))
	}
	if result[0]["id"] != "doc1" {
		t.Errorf("ReciprocalRankFusion() результат = %v, ожидался doc1", result[0]["id"])
	}
}

func TestFusionResultStruct(t *testing.T) {
	result := &FusionResult{
		ID:    "test_doc",
		Score: 0.95,
		Payload: map[string]interface{}{
			"text": "тестовый документ",
			"page": 1,
		},
	}

	if result.ID != "test_doc" {
		t.Errorf("FusionResult.ID = %s, ожидался test_doc", result.ID)
	}
	if result.Score != 0.95 {
		t.Errorf("FusionResult.Score = %f, ожидалось 0.95", result.Score)
	}
	if result.Payload["text"] != "тестовый документ" {
		t.Errorf("FusionResult.Payload[text] = %s, ожидался 'тестовый документ'", result.Payload["text"])
	}
}