package eval

import (
	"os"
	"testing"
)

func TestLoadQuestions(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "questions_*.jsonl")
	if err != nil {
		t.Fatalf("Не удалось создать временный файл: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	content := `{"query":"Что такое RAG?","expected_docs":["doc1.md"]}
{"query":"Что такое эмбеддинги?","expected_docs":["doc2.md"]}`
	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("Не удалось записать в файл: %v", err)
	}
	tmpFile.Close()

	questions, err := loadQuestions(tmpFile.Name())
	if err != nil {
		t.Errorf("loadQuestions() ошибка: %v", err)
	}
	if len(questions) != 2 {
		t.Errorf("loadQuestions() вернул %d вопросов, ожидалось 2", len(questions))
	}
}

func TestCalcRecall(t *testing.T) {
	tests := []struct {
		name         string
		foundDocs    []string
		expectedDocs []string
		k            int
		expected     float64
	}{
		{
			name:         "полное совпадение",
			foundDocs:    []string{"doc1.md", "doc2.md"},
			expectedDocs: []string{"doc1.md", "doc2.md"},
			k:            5,
			expected:     1.0,
		},
		{
			name:         "частичное совпадение",
			foundDocs:    []string{"doc1.md", "doc3.md"},
			expectedDocs: []string{"doc1.md", "doc2.md"},
			k:            5,
			expected:     0.5,
		},
		{
			name:         "пустые результаты",
			foundDocs:    []string{},
			expectedDocs: []string{"doc1.md"},
			k:            5,
			expected:     0.0,
		},
		{
			name:         "нет ожидаемых",
			foundDocs:    []string{"doc1.md"},
			expectedDocs: []string{},
			k:            5,
			expected:     0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calcRecall(tt.foundDocs, tt.expectedDocs, tt.k)
			if result != tt.expected {
				t.Errorf("calcRecall() = %f, ожидалось %f", result, tt.expected)
			}
		})
	}
}

func TestCalcMRR(t *testing.T) {
	tests := []struct {
		name         string
		foundDocs    []string
		expectedDocs []string
		expected     float64
	}{
		{
			name:         "первый правильный",
			foundDocs:    []string{"doc1.md", "doc2.md"},
			expectedDocs: []string{"doc1.md"},
			expected:     1.0,
		},
		{
			name:         "второй правильный",
			foundDocs:    []string{"doc3.md", "doc1.md"},
			expectedDocs: []string{"doc1.md"},
			expected:     0.5,
		},
		{
			name:         "не найден",
			foundDocs:    []string{"doc3.md", "doc4.md"},
			expectedDocs: []string{"doc1.md"},
			expected:     0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calcMRR(tt.foundDocs, tt.expectedDocs)
			if result != tt.expected {
				t.Errorf("calcMRR() = %f, ожидалось %f", result, tt.expected)
			}
		})
	}
}